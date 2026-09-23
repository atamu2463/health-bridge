package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/netip"
	"sort"
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
	limit       int
	windowStart time.Time
	lastSeen    time.Time
}

type loginRateLimitKey struct {
	digest [sha256.Size]byte
	limit  int
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
	keys := []loginRateLimitKey{
		{
			digest: rateLimitDigest(rateLimitIdentifierIP, normalizedClientIP(clientIP)),
			limit:  limiter.ipLimit,
		},
	}
	if email != "" {
		keys = append(keys, loginRateLimitKey{
			digest: rateLimitDigest(rateLimitIdentifierEmail, email),
			limit:  limiter.emailLimit,
		})
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	limiter.cleanupExpired(now)
	missingKeys := 0
	protectedKeys := make(map[[sha256.Size]byte]struct{}, len(keys))
	for _, key := range keys {
		protectedKeys[key.digest] = struct{}{}
		entry, exists := limiter.entries[key.digest]
		if exists && now.Sub(entry.windowStart) < limiter.window && entry.count >= key.limit {
			return false
		}
		if !exists || now.Sub(entry.windowStart) >= limiter.window {
			missingKeys++
		}
	}
	if missingKeys > 0 && !limiter.ensureCapacity(now, missingKeys, protectedKeys) {
		return false
	}

	for _, key := range keys {
		entry, exists := limiter.entries[key.digest]
		if !exists {
			entry = loginRateLimitEntry{
				limit:       key.limit,
				windowStart: now,
			}
		}
		entry.count++
		entry.lastSeen = now
		limiter.entries[key.digest] = entry
	}

	return true
}

func (limiter *loginRateLimiter) cleanupExpired(now time.Time) {
	if !limiter.nextCleanup.IsZero() && now.Before(limiter.nextCleanup) {
		return
	}
	limiter.removeExpired(now)
	limiter.nextCleanup = now.Add(limiter.window)
}

func (limiter *loginRateLimiter) removeExpired(now time.Time) {
	for key, entry := range limiter.entries {
		if now.Sub(entry.windowStart) >= limiter.window {
			delete(limiter.entries, key)
		}
	}
}

func (limiter *loginRateLimiter) ensureCapacity(now time.Time, needed int, protectedKeys map[[sha256.Size]byte]struct{}) bool {
	limiter.removeExpired(now)
	available := limiter.maxKeys - len(limiter.entries)
	if available >= needed {
		return true
	}

	candidates := make([][sha256.Size]byte, 0, len(limiter.entries))
	for key, entry := range limiter.entries {
		if _, protected := protectedKeys[key]; protected || entry.count >= entry.limit {
			continue
		}
		candidates = append(candidates, key)
	}
	sort.Slice(candidates, func(left, right int) bool {
		return limiter.entries[candidates[left]].lastSeen.Before(limiter.entries[candidates[right]].lastSeen)
	})

	toRemove := needed - available
	if len(candidates) < toRemove {
		return false
	}
	for _, key := range candidates[:toRemove] {
		delete(limiter.entries, key)
	}
	return true
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
