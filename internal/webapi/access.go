package webapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"steam-profit-calculator/internal/observability"
)

const (
	minimumDemoPasswordRunes = 12
	maximumDemoUsernameRunes = 128
	maximumDemoPasswordRunes = 1024
	maximumAccessLimit       = 100000
	defaultTrackedClients    = 4096
)

// AccessOptions configures the optional single-user hosted-demo boundary.
// Zero values preserve the unprotected local-development behavior.
type AccessOptions struct {
	Username               string
	Password               string
	RequestsPerMinute      int
	AssistantRunsPerHour   int
	TrustProxyClientHeader bool
}

// AccessOptionsFromEnvironment loads hosted-demo controls without exposing secrets.
func AccessOptionsFromEnvironment() (AccessOptions, error) {
	requestsPerMinute, err := optionalNonNegativeEnvironmentInteger("STEAM_PROFIT_RATE_LIMIT_PER_MINUTE")
	if err != nil {
		return AccessOptions{}, err
	}
	assistantRunsPerHour, err := optionalNonNegativeEnvironmentInteger("STEAM_PROFIT_AI_RUNS_PER_HOUR")
	if err != nil {
		return AccessOptions{}, err
	}
	trustProxy, err := optionalEnvironmentBoolean("STEAM_PROFIT_TRUST_PROXY_HEADERS")
	if err != nil {
		return AccessOptions{}, err
	}
	options := AccessOptions{
		Username:               strings.TrimSpace(os.Getenv("STEAM_PROFIT_DEMO_USERNAME")),
		Password:               os.Getenv("STEAM_PROFIT_DEMO_PASSWORD"),
		RequestsPerMinute:      requestsPerMinute,
		AssistantRunsPerHour:   assistantRunsPerHour,
		TrustProxyClientHeader: trustProxy,
	}
	if err := options.validate(); err != nil {
		return AccessOptions{}, err
	}
	return options, nil
}

func (o AccessOptions) validate() error {
	hasUsername := o.Username != ""
	hasPassword := o.Password != ""
	if hasUsername != hasPassword {
		return fmt.Errorf("demo access requires both STEAM_PROFIT_DEMO_USERNAME and STEAM_PROFIT_DEMO_PASSWORD")
	}
	if strings.ContainsAny(o.Username, ":\r\n") {
		return fmt.Errorf("STEAM_PROFIT_DEMO_USERNAME must not contain a colon or line break")
	}
	if len([]rune(o.Username)) > maximumDemoUsernameRunes {
		return fmt.Errorf("STEAM_PROFIT_DEMO_USERNAME must contain at most %d characters", maximumDemoUsernameRunes)
	}
	if strings.ContainsAny(o.Password, "\r\n") {
		return fmt.Errorf("STEAM_PROFIT_DEMO_PASSWORD must not contain a line break")
	}
	if hasPassword && len([]rune(o.Password)) < minimumDemoPasswordRunes {
		return fmt.Errorf("STEAM_PROFIT_DEMO_PASSWORD must contain at least %d characters", minimumDemoPasswordRunes)
	}
	if len([]rune(o.Password)) > maximumDemoPasswordRunes {
		return fmt.Errorf("STEAM_PROFIT_DEMO_PASSWORD must contain at most %d characters", maximumDemoPasswordRunes)
	}
	if o.RequestsPerMinute < 0 || o.RequestsPerMinute > maximumAccessLimit {
		return fmt.Errorf("STEAM_PROFIT_RATE_LIMIT_PER_MINUTE must be between 0 and %d", maximumAccessLimit)
	}
	if o.AssistantRunsPerHour < 0 || o.AssistantRunsPerHour > maximumAccessLimit {
		return fmt.Errorf("STEAM_PROFIT_AI_RUNS_PER_HOUR must be between 0 and %d", maximumAccessLimit)
	}
	return nil
}

type accessWindow struct {
	started  time.Time
	lastSeen time.Time
	count    int
}

type accessController struct {
	options    AccessOptions
	metrics    *observability.Metrics
	now        func() time.Time
	maxClients int

	mu              sync.Mutex
	clients         map[string]accessWindow
	assistantWindow accessWindow
}

func newAccessController(options AccessOptions, metrics *observability.Metrics) (*accessController, error) {
	return newAccessControllerWithClock(options, metrics, time.Now, defaultTrackedClients)
}

func newAccessControllerWithClock(options AccessOptions, metrics *observability.Metrics, now func() time.Time, maxClients int) (*accessController, error) {
	if err := options.validate(); err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	if maxClients <= 0 {
		maxClients = defaultTrackedClients
	}
	return &accessController{
		options: options, metrics: metrics, now: now, maxClients: maxClients,
		clients: make(map[string]accessWindow),
	}, nil
}

func (c *accessController) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		if c.authenticationEnabled() && !c.authorized(r) {
			c.metrics.AccessUnauthorized()
			w.Header().Set("WWW-Authenticate", `Basic realm="Steam Profit Demo", charset="UTF-8"`)
			writeAPIError(w, http.StatusUnauthorized, "authentication_required", "demo access credentials are required")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && c.options.RequestsPerMinute > 0 {
			allowed, retryAfter := c.allowAPIRequest(c.clientKey(r))
			if !allowed {
				c.metrics.AccessRateLimited()
				writeRetryAfter(w, retryAfter)
				writeAPIError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "API request limit exceeded; retry later")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (c *accessController) authenticationEnabled() bool {
	return c != nil && c.options.Username != "" && c.options.Password != ""
}

func (c *accessController) authorized(r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}
	usernameMatches := constantTimeCredentialMatch(username, c.options.Username)
	passwordMatches := constantTimeCredentialMatch(password, c.options.Password)
	return usernameMatches&passwordMatches == 1
}

func constantTimeCredentialMatch(candidate, expected string) int {
	candidateDigest := sha256.Sum256([]byte(candidate))
	expectedDigest := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(candidateDigest[:], expectedDigest[:])
}

func (c *accessController) allowAPIRequest(client string) (bool, time.Duration) {
	if c == nil || c.options.RequestsPerMinute <= 0 {
		return true, 0
	}
	now := c.now().UTC()
	c.mu.Lock()
	defer c.mu.Unlock()

	window, exists := c.clients[client]
	if !exists {
		c.ensureClientCapacity(now)
		window = accessWindow{started: now}
	}
	if now.Before(window.started) || now.Sub(window.started) >= time.Minute {
		window = accessWindow{started: now}
	}
	window.lastSeen = now
	if window.count >= c.options.RequestsPerMinute {
		c.clients[client] = window
		return false, boundedRetryAfter(window.started.Add(time.Minute).Sub(now))
	}
	window.count++
	c.clients[client] = window
	return true, 0
}

func (c *accessController) allowAssistantRun() (bool, time.Duration) {
	if c == nil || c.options.AssistantRunsPerHour <= 0 {
		return true, 0
	}
	now := c.now().UTC()
	c.mu.Lock()
	defer c.mu.Unlock()
	window := c.assistantWindow
	if window.started.IsZero() || now.Before(window.started) || now.Sub(window.started) >= time.Hour {
		window = accessWindow{started: now}
	}
	window.lastSeen = now
	if window.count >= c.options.AssistantRunsPerHour {
		c.assistantWindow = window
		return false, boundedRetryAfter(window.started.Add(time.Hour).Sub(now))
	}
	window.count++
	c.assistantWindow = window
	return true, 0
}

func (c *accessController) ensureClientCapacity(now time.Time) {
	for key, window := range c.clients {
		if now.Sub(window.lastSeen) >= 2*time.Minute || now.Before(window.lastSeen) {
			delete(c.clients, key)
		}
	}
	if len(c.clients) < c.maxClients {
		return
	}
	var oldestKey string
	var oldestTime time.Time
	for key, window := range c.clients {
		if oldestKey == "" || window.lastSeen.Before(oldestTime) {
			oldestKey = key
			oldestTime = window.lastSeen
		}
	}
	if oldestKey != "" {
		delete(c.clients, oldestKey)
	}
}

func (c *accessController) clientKey(r *http.Request) string {
	if c != nil && c.options.TrustProxyClientHeader {
		candidates := []string{
			strings.TrimSpace(r.Header.Get("X-Real-IP")),
			strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]),
		}
		for _, candidate := range candidates {
			if parsed := net.ParseIP(candidate); parsed != nil {
				return parsed.String()
			}
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		if parsed := net.ParseIP(host); parsed != nil {
			return parsed.String()
		}
		return host
	}
	if parsed := net.ParseIP(strings.TrimSpace(r.RemoteAddr)); parsed != nil {
		return parsed.String()
	}
	return "unknown"
}

func optionalNonNegativeEnvironmentInteger(name string) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 || value > maximumAccessLimit {
		return 0, fmt.Errorf("%s must be an integer between 0 and %d", name, maximumAccessLimit)
	}
	return value, nil
}

func optionalEnvironmentBoolean(name string) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", name)
	}
	return value, nil
}

func boundedRetryAfter(duration time.Duration) time.Duration {
	if duration <= 0 {
		return time.Second
	}
	return duration
}

func writeRetryAfter(w http.ResponseWriter, duration time.Duration) {
	seconds := int64((duration + time.Second - 1) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.FormatInt(seconds, 10))
}
