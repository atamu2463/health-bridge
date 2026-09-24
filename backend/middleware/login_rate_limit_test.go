package middleware

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"backend/api"

	"github.com/gin-gonic/gin"
)

type countingReadCloser struct {
	reader    io.Reader
	bytesRead int
}

func (reader *countingReadCloser) Read(buffer []byte) (int, error) {
	read, err := reader.reader.Read(buffer)
	reader.bytesRead += read
	return read, err
}

func (*countingReadCloser) Close() error {
	return nil
}

func TestLoginRateLimitAllowsLimitsAndAllowsAfterWindow(t *testing.T) {
	currentTime := time.Date(2026, time.September, 23, 12, 0, 0, 0, time.UTC)
	limiter := newLoginRateLimiter(loginRateLimitConfig{
		ipLimit:    2,
		emailLimit: 2,
		window:     time.Minute,
		maxKeys:    100,
		now:        func() time.Time { return currentTime },
	})

	nextCalled := 0
	router := loginRateLimitTestRouter(limiter, &nextCalled)
	for attempt := 1; attempt <= 3; attempt++ {
		response := performLoginRateLimitRequest(router, "192.0.2.10:1234", "employee@example.invalid")
		if attempt <= 2 {
			if response.Code != http.StatusNoContent {
				t.Fatalf("attempt %d status = %d, want %d", attempt, response.Code, http.StatusNoContent)
			}
			continue
		}
		if response.Code != http.StatusTooManyRequests {
			t.Fatalf("attempt %d status = %d, want %d", attempt, response.Code, http.StatusTooManyRequests)
		}
		wantBody := `{"error":{"code":"` + api.ErrorCodeTooManyRequests + `","message":"しばらく待ってから再度お試しください"}}`
		if body := strings.TrimSpace(response.Body.String()); body != wantBody {
			t.Fatalf("response body = %q, want %q", body, wantBody)
		}
	}
	if nextCalled != 2 {
		t.Fatalf("handler calls = %d, want 2", nextCalled)
	}

	currentTime = currentTime.Add(time.Minute)
	response := performLoginRateLimitRequest(router, "192.0.2.10:1234", "employee@example.invalid")
	if response.Code != http.StatusNoContent {
		t.Fatalf("status after window = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestLoginRateLimitUsesNormalizedEmailAcrossClientIPs(t *testing.T) {
	limiter := newLoginRateLimiter(loginRateLimitConfig{
		ipLimit:    10,
		emailLimit: 1,
		window:     time.Minute,
		maxKeys:    100,
		now:        time.Now,
	})
	router := loginRateLimitTestRouter(limiter, new(int))

	first := performLoginRateLimitRequest(router, "192.0.2.10:1234", " Employee@Example.Invalid ")
	second := performLoginRateLimitRequest(router, "192.0.2.11:1234", "employee@example.invalid")
	if first.Code != http.StatusNoContent || second.Code != http.StatusTooManyRequests {
		t.Fatalf("statuses = %d, %d; want %d, %d", first.Code, second.Code, http.StatusNoContent, http.StatusTooManyRequests)
	}
}

func TestLoginRateLimitUsesClientIPAcrossEmailAddresses(t *testing.T) {
	limiter := newLoginRateLimiter(loginRateLimitConfig{
		ipLimit:    1,
		emailLimit: 10,
		window:     time.Minute,
		maxKeys:    100,
		now:        time.Now,
	})
	router := loginRateLimitTestRouter(limiter, new(int))

	first := performLoginRateLimitRequest(router, "192.0.2.10:1234", "first@example.invalid")
	second := performLoginRateLimitRequest(router, "192.0.2.10:5678", "second@example.invalid")
	if first.Code != http.StatusNoContent || second.Code != http.StatusTooManyRequests {
		t.Fatalf("statuses = %d, %d; want %d, %d", first.Code, second.Code, http.StatusNoContent, http.StatusTooManyRequests)
	}
}

func TestLoginRateLimiterBoundsStoredKeys(t *testing.T) {
	limiter := newLoginRateLimiter(loginRateLimitConfig{
		ipLimit:    10,
		emailLimit: 10,
		window:     time.Minute,
		maxKeys:    3,
		now:        time.Now,
	})

	for index := 1; index <= 10; index++ {
		limiter.allow(fmt.Sprintf("192.0.2.%d", index), fmt.Sprintf("employee%d@example.invalid", index))
	}
	if len(limiter.entries) > limiter.maxKeys {
		t.Fatalf("stored keys = %d, want at most %d", len(limiter.entries), limiter.maxKeys)
	}
}

func TestLoginRateLimiterDoesNotEvictBlockedEmailAtCapacity(t *testing.T) {
	currentTime := time.Date(2026, time.September, 24, 12, 0, 0, 0, time.UTC)
	limiter := newLoginRateLimiter(loginRateLimitConfig{
		ipLimit:    100,
		emailLimit: 2,
		window:     time.Minute,
		maxKeys:    3,
		now:        func() time.Time { return currentTime },
	})
	blockedEmail := "blocked@example.invalid"

	if !limiter.allow("192.0.2.1", blockedEmail) || !limiter.allow("192.0.2.1", blockedEmail) {
		t.Fatal("requests up to the email limit should be allowed")
	}
	if !limiter.allow("192.0.2.2", "second@example.invalid") ||
		!limiter.allow("192.0.2.3", "third@example.invalid") {
		t.Fatal("new keys should use capacity by evicting eligible entries")
	}
	if limiter.allow("192.0.2.4", blockedEmail) {
		t.Fatal("blocked email was reset by capacity eviction")
	}
	assertRateLimitEntryCount(t, limiter, rateLimitIdentifierEmail, blockedEmail, 2)
}

func TestLoginRateLimiterDoesNotEvictBlockedIPAtCapacity(t *testing.T) {
	currentTime := time.Date(2026, time.September, 24, 12, 0, 0, 0, time.UTC)
	limiter := newLoginRateLimiter(loginRateLimitConfig{
		ipLimit:    2,
		emailLimit: 100,
		window:     time.Minute,
		maxKeys:    3,
		now:        func() time.Time { return currentTime },
	})
	blockedIP := "192.0.2.1"

	if !limiter.allow(blockedIP, "first@example.invalid") ||
		!limiter.allow(blockedIP, "second@example.invalid") {
		t.Fatal("requests up to the IP limit should be allowed")
	}
	if !limiter.allow("192.0.2.2", "third@example.invalid") {
		t.Fatal("new keys should use capacity by evicting eligible entries")
	}
	if limiter.allow(blockedIP, "fourth@example.invalid") {
		t.Fatal("blocked IP was reset by capacity eviction")
	}
	assertRateLimitEntryCount(t, limiter, rateLimitIdentifierIP, blockedIP, 2)
}

func TestLoginRateLimiterDoesNotPartiallyUpdateWhenCapacityCannotBeSecured(t *testing.T) {
	currentTime := time.Date(2026, time.September, 24, 12, 0, 0, 0, time.UTC)
	limiter := newLoginRateLimiter(loginRateLimitConfig{
		ipLimit:    3,
		emailLimit: 1,
		window:     time.Minute,
		maxKeys:    2,
		now:        func() time.Time { return currentTime },
	})
	clientIP := "192.0.2.1"

	if !limiter.allow(clientIP, "blocked@example.invalid") {
		t.Fatal("initial request should be allowed")
	}
	if limiter.allow(clientIP, "new@example.invalid") {
		t.Fatal("request should be rejected when capacity cannot be secured")
	}
	assertRateLimitEntryCount(t, limiter, rateLimitIdentifierIP, clientIP, 1)
	if _, exists := limiter.entries[rateLimitDigest(rateLimitIdentifierEmail, "new@example.invalid")]; exists {
		t.Fatal("new email counter was partially inserted")
	}
}

func TestLoginRateLimiterReusesCapacityFromExpiredEntries(t *testing.T) {
	currentTime := time.Date(2026, time.September, 24, 12, 0, 0, 0, time.UTC)
	limiter := newLoginRateLimiter(loginRateLimitConfig{
		ipLimit:    10,
		emailLimit: 10,
		window:     time.Minute,
		maxKeys:    2,
		now:        func() time.Time { return currentTime },
	})
	oldIP := "192.0.2.1"
	oldEmail := "old@example.invalid"

	if !limiter.allow(oldIP, oldEmail) {
		t.Fatal("initial request should be allowed")
	}
	currentTime = currentTime.Add(time.Minute)
	// 定期cleanupがまだ走らない状態でも、容量確保時には期限切れを優先して削除する。
	limiter.nextCleanup = currentTime.Add(time.Minute)
	if !limiter.allow("192.0.2.2", "new@example.invalid") {
		t.Fatal("expired entries should make capacity reusable")
	}
	if _, exists := limiter.entries[rateLimitDigest(rateLimitIdentifierIP, oldIP)]; exists {
		t.Fatal("expired IP entry was not removed")
	}
	if _, exists := limiter.entries[rateLimitDigest(rateLimitIdentifierEmail, oldEmail)]; exists {
		t.Fatal("expired email entry was not removed")
	}
}

func TestLoginRateLimiterEvictsOldestEligibleEntries(t *testing.T) {
	currentTime := time.Date(2026, time.September, 24, 12, 0, 0, 0, time.UTC)
	limiter := newLoginRateLimiter(loginRateLimitConfig{
		ipLimit:    100,
		emailLimit: 100,
		window:     time.Minute,
		maxKeys:    3,
		now:        func() time.Time { return currentTime },
	})
	oldIP := "192.0.2.1"
	oldEmail := "old@example.invalid"

	if !limiter.allow(oldIP, oldEmail) {
		t.Fatal("initial request should be allowed")
	}
	currentTime = currentTime.Add(time.Second)
	newerIP := "192.0.2.2"
	if !limiter.allow(newerIP, "") {
		t.Fatal("newer IP should be stored")
	}
	currentTime = currentTime.Add(time.Second)
	if !limiter.allow("192.0.2.3", "new@example.invalid") {
		t.Fatal("request should evict eligible entries")
	}
	if _, exists := limiter.entries[rateLimitDigest(rateLimitIdentifierIP, newerIP)]; !exists {
		t.Fatal("newer eligible entry was evicted before older entries")
	}
	if _, exists := limiter.entries[rateLimitDigest(rateLimitIdentifierIP, oldIP)]; exists {
		t.Fatal("oldest IP entry was not evicted")
	}
	if _, exists := limiter.entries[rateLimitDigest(rateLimitIdentifierEmail, oldEmail)]; exists {
		t.Fatal("oldest email entry was not evicted")
	}
}

func TestNormalizedLoginEmailLimitsBodyReadAndRestoresBody(t *testing.T) {
	body := strings.Repeat("x", maxRateLimitRequestBody+1024)
	trackedBody := &countingReadCloser{reader: strings.NewReader(body)}
	request := httptest.NewRequest(http.MethodPost, "/login", nil)
	request.Body = trackedBody
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	if email := normalizedLoginEmail(context); email != "" {
		t.Fatalf("email = %q, want empty", email)
	}
	if trackedBody.bytesRead != maxRateLimitRequestBody+1 {
		t.Fatalf("bytes read = %d, want %d", trackedBody.bytesRead, maxRateLimitRequestBody+1)
	}
	restoredBody, err := io.ReadAll(context.Request.Body)
	if err != nil {
		t.Fatalf("restored body read failed: %v", err)
	}
	if string(restoredBody) != body {
		t.Fatal("request body was not fully restored")
	}
}

func TestLoginRateLimiterIsSafeForConcurrentAccess(t *testing.T) {
	limiter := newLoginRateLimiter(loginRateLimitConfig{
		ipLimit:    1000,
		emailLimit: 1000,
		window:     time.Minute,
		maxKeys:    100,
		now:        time.Now,
	})

	const requests = 100
	var waitGroup sync.WaitGroup
	waitGroup.Add(requests)
	for index := 0; index < requests; index++ {
		go func() {
			defer waitGroup.Done()
			if !limiter.allow("192.0.2.10", "employee@example.invalid") {
				t.Error("request was unexpectedly rate limited")
			}
		}()
	}
	waitGroup.Wait()

	if len(limiter.entries) != 2 {
		t.Fatalf("stored keys = %d, want 2", len(limiter.entries))
	}
}

func assertRateLimitEntryCount(t *testing.T, limiter *loginRateLimiter, identifierType, value string, want int) {
	t.Helper()
	entry, exists := limiter.entries[rateLimitDigest(identifierType, value)]
	if !exists {
		t.Fatalf("%s entry does not exist", identifierType)
	}
	if entry.count != want {
		t.Fatalf("%s count = %d, want %d", identifierType, entry.count, want)
	}
}

func loginRateLimitTestRouter(limiter *loginRateLimiter, nextCalled *int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		panic(err)
	}
	router.POST("/login", limiter.middleware(), func(c *gin.Context) {
		(*nextCalled)++
		var request struct {
			Email string `json:"email"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusNoContent)
	})
	return router
}

func performLoginRateLimitRequest(router *gin.Engine, remoteAddress, email string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"`+email+`","password":"secret"}`))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = remoteAddress
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
