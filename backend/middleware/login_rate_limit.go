package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/netip"
	"strings"
	"sync"
	"time"

	"backend/api"

	"github.com/gin-gonic/gin"
)

const (
	defaultLoginIPLimit      = 30
	defaultLoginEmailLimit   = 10
	defaultLoginRateWindow   = time.Minute
	defaultLoginRateMaxKeys  = 10_000
	maxRateLimitRequestBody  = 64 << 10
	rateLimitIdentifierIP    = "ip"
	rateLimitIdentifierEmail = "email"
)

type loginRateLimitConfig struct {
	ipLimit    int
	emailLimit int
	window     time.Duration
	maxKeys    int
	now        func() time.Time
}

type loginRateLimitEntry struct {
	count       int
	windowStart time.Time
	lastSeen    time.Time
}

type loginRateLimiter struct {
	mu          sync.Mutex
	entries     map[[sha256.Size]byte]loginRateLimitEntry
	ipLimit     int
	emailLimit  int
	window      time.Duration
	maxKeys     int
	now         func() time.Time
	nextCleanup time.Time
}

type replayableBody struct {
	io.Reader
	io.Closer
}

func LoginRateLimit() gin.HandlerFunc {
	return newLoginRateLimiter(loginRateLimitConfig{
		ipLimit:    defaultLoginIPLimit,
		emailLimit: defaultLoginEmailLimit,
		window:     defaultLoginRateWindow,
		maxKeys:    defaultLoginRateMaxKeys,
		now:        time.Now,
	}).middleware()
}

func newLoginRateLimiter(config loginRateLimitConfig) *loginRateLimiter {
	return &loginRateLimiter{
		entries:    make(map[[sha256.Size]byte]loginRateLimitEntry),
		ipLimit:    config.ipLimit,
		emailLimit: config.emailLimit,
		window:     config.window,
		maxKeys:    config.maxKeys,
		now:        config.now,
	}
}

func (limiter *loginRateLimiter) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		email := normalizedLoginEmail(c)
		if !limiter.allow(c.ClientIP(), email) {
			api.TooManyRequests(c)
			return
		}

		c.Next()
	}
}

func (limiter *loginRateLimiter) allow(clientIP, email string) bool {
	now := limiter.now()
	keys := []struct {
		value string
		limit int
	}{
		{value: normalizedClientIP(clientIP), limit: limiter.ipLimit},
	}
	if email != "" {
		keys = append(keys, struct {
			value string
			limit int
		}{value: email, limit: limiter.emailLimit})
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	limiter.cleanupExpired(now)
	for index, key := range keys {
		identifierType := rateLimitIdentifierIP
		if index == 1 {
			identifierType = rateLimitIdentifierEmail
		}
		digest := rateLimitDigest(identifierType, key.value)
		entry, exists := limiter.entries[digest]
		if exists && now.Sub(entry.windowStart) < limiter.window && entry.count >= key.limit {
			return false
		}
	}

	for index := range keys {
		identifierType := rateLimitIdentifierIP
		if index == 1 {
			identifierType = rateLimitIdentifierEmail
		}
		digest := rateLimitDigest(identifierType, keys[index].value)
		entry, exists := limiter.entries[digest]
		if !exists || now.Sub(entry.windowStart) >= limiter.window {
			if !exists {
				limiter.makeRoom()
			}
			entry = loginRateLimitEntry{windowStart: now}
		}
		entry.count++
		entry.lastSeen = now
		limiter.entries[digest] = entry
	}

	return true
}

func (limiter *loginRateLimiter) cleanupExpired(now time.Time) {
	if !limiter.nextCleanup.IsZero() && now.Before(limiter.nextCleanup) {
		return
	}
	for key, entry := range limiter.entries {
		if now.Sub(entry.windowStart) >= limiter.window {
			delete(limiter.entries, key)
		}
	}
	limiter.nextCleanup = now.Add(limiter.window)
}

func (limiter *loginRateLimiter) makeRoom() {
	if len(limiter.entries) < limiter.maxKeys {
		return
	}

	var oldestKey [sha256.Size]byte
	var oldestTime time.Time
	for key, entry := range limiter.entries {
		if oldestTime.IsZero() || entry.lastSeen.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.lastSeen
		}
	}
	delete(limiter.entries, oldestKey)
}

func normalizedLoginEmail(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	originalBody := c.Request.Body
	body, err := io.ReadAll(io.LimitReader(originalBody, maxRateLimitRequestBody+1))
	c.Request.Body = replayableBody{
		Reader: io.MultiReader(bytes.NewReader(body), originalBody),
		Closer: originalBody,
	}
	if err != nil || len(body) > maxRateLimitRequestBody {
		return ""
	}

	var request struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(request.Email))
}

func normalizedClientIP(value string) string {
	address, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil {
		return strings.TrimSpace(value)
	}
	return address.Unmap().String()
}

func rateLimitDigest(identifierType, value string) [sha256.Size]byte {
	return sha256.Sum256([]byte(identifierType + "\x00" + value))
}
