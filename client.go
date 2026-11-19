package mpesa

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Version is the version of this module.
const Version = "1.0.0"

const (
	// SandboxBaseURL is the base URL of the Daraja sandbox.
	SandboxBaseURL = "https://sandbox.safaricom.co.ke"
	// ProductionBaseURL is the base URL of the live Daraja API.
	ProductionBaseURL = "https://api.safaricom.co.ke"

	// pathOAuth is the token endpoint. Safaricom wants the grant type as a
	// query parameter and credentials in the Authorization header.
	pathOAuth = "/oauth/v1/generate?grant_type=client_credentials"

	defaultTimeout      = 30 * time.Second
	defaultRetryBackoff = 300 * time.Millisecond
	maxRetryBackoff     = 5 * time.Second
	maxResponseBytes    = 1 << 20 // 1 MiB
)

// Env identifies the Daraja environment a Client talks to.
type Env string

const (
	// DEV targets the Safaricom sandbox. The zero value of Env is treated as
	// DEV.
	DEV Env = "dev"
	// PRODUCTION targets the live Safaricom API.
	PRODUCTION Env = "production"
)

// baseURL resolves the API host for the environment.
func (e Env) baseURL() (string, error) {
	switch e {
	case PRODUCTION:
		return ProductionBaseURL, nil
	case DEV, "":
		return SandboxBaseURL, nil
	default:
		return "", fmt.Errorf("%w: %q (use mpesa.DEV or mpesa.PRODUCTION)", ErrInvalidEnvironment, string(e))
	}
}

// Client is a Safaricom Daraja (M-Pesa) API client. It is safe for concurrent
// use by multiple goroutines.
type Client struct {
	appKey    string
	appSecret string
	env       Env

	baseURL      string
	http         *http.Client
	passKey      string
	shortCode    string
	certificate  *x509.Certificate
	initiatorPwd string
	maxRetries   int
	retryBackoff time.Duration
	userAgent    string
	now          func() time.Time

	// tokenMu guards the cached OAuth token. It is held for the duration of a
	// token request so that concurrent callers only fetch one token.
	tokenMu     sync.Mutex
	token       string
	tokenExpiry time.Time
}

// Mpesa is the historical name of Client.
//
// Deprecated: use Client.
type Mpesa = Client

// Option configures a Client. Options are applied in order by New.
type Option func(*Client) error

// New returns a Client for the given app key, app secret and environment.
//
// The app key and app secret are the consumer key/secret issued by Safaricom.
// Use WithPassKey and WithShortCode to let the client build STK push passwords
// automatically.
func New(appKey, appSecret string, env Env, opts ...Option) (*Client, error) {
	appKey = strings.TrimSpace(appKey)
	appSecret = strings.TrimSpace(appSecret)
	if appKey == "" || appSecret == "" {
		return nil, ErrMissingCredentials
	}

	baseURL, err := env.baseURL()
	if err != nil {
		return nil, err
	}
	if env == "" {
		env = DEV
	}

	c := &Client{
		appKey:       appKey,
		appSecret:    appSecret,
		env:          env,
		baseURL:      baseURL,
		http:         &http.Client{Timeout: defaultTimeout},
		retryBackoff: defaultRetryBackoff,
		userAgent:    "mpesa-api-go/" + Version,
		now:          time.Now,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	if c.http == nil {
		c.http = &http.Client{Timeout: defaultTimeout}
	}
	if c.maxRetries < 0 {
		c.maxRetries = 0
	}
	return c, nil
}

// WithHTTPClient sets the HTTP client used for API calls. The client is used
// as is, so its timeout and transport configuration apply.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc == nil {
			return errors.New("mpesa: http client must not be nil")
		}
		c.http = hc
		return nil
	}
}

// WithTimeout sets the request timeout on a copy of the configured HTTP client.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) error {
		if d <= 0 {
			return errors.New("mpesa: timeout must be greater than zero")
		}
		if c.http == nil {
			c.http = &http.Client{}
		}
		clone := *c.http
		clone.Timeout = d
		c.http = &clone
		return nil
	}
}

// WithBaseURL overrides the API base URL, for example to point at a mock
// server in tests.
func WithBaseURL(rawURL string) Option {
	return func(c *Client) error {
		parsed, err := url.Parse(strings.TrimSpace(rawURL))
		if err != nil {
			return fmt.Errorf("mpesa: invalid base URL: %w", err)
		}
		if !parsed.IsAbs() || parsed.Host == "" {
			return fmt.Errorf("mpesa: invalid base URL %q: an absolute URL is required", rawURL)
		}
		switch strings.ToLower(parsed.Scheme) {
		case "http", "https":
		default:
			return fmt.Errorf("mpesa: invalid base URL %q: scheme must be http or https", rawURL)
		}
		c.baseURL = strings.TrimRight(parsed.String(), "/")
		return nil
	}
}

// WithPassKey sets the Lipa na M-Pesa pass key used to build STK push
// passwords.
func WithPassKey(passKey string) Option {
	return func(c *Client) error {
		c.passKey = strings.TrimSpace(passKey)
		return nil
	}
}

// WithShortCode sets the default business short code used by requests that do
// not specify one.
func WithShortCode(shortCode string) Option {
	return func(c *Client) error {
		c.shortCode = strings.TrimSpace(shortCode)
		return nil
	}
}

// WithCertificate sets the Safaricom RSA certificate used to generate security
// credentials for B2C, B2B, reversal, balance and transaction status requests.
// Download the sandbox or production certificate from the Daraja portal and
// load it with LoadCertificate.
func WithCertificate(cert *x509.Certificate) Option {
	return func(c *Client) error {
		if cert == nil {
			return errors.New("mpesa: certificate must not be nil")
		}
		c.certificate = cert
		return nil
	}
}

// WithInitiatorPassword sets the initiator password that is RSA encrypted into
// a security credential when a request does not provide one.
func WithInitiatorPassword(password string) Option {
	return func(c *Client) error {
		c.initiatorPwd = password
		return nil
	}
}

// WithRetries sets how many times a request is retried when Daraja answers with
// a retryable gateway/rate limit error (429, 502, 503, 504). Retries are
// disabled by default because payment requests must not be replayed blindly.
func WithRetries(n int) Option {
	return func(c *Client) error {
		if n < 0 {
			return errors.New("mpesa: retries must not be negative")
		}
		c.maxRetries = n
		return nil
	}
}

// WithRetryBackoff sets the base delay between retries. The delay doubles on
// every attempt and is jittered.
func WithRetryBackoff(d time.Duration) Option {
	return func(c *Client) error {
		if d < 0 {
			return errors.New("mpesa: retry backoff must not be negative")
		}
		c.retryBackoff = d
		return nil
	}
}

// WithUserAgent overrides the User-Agent header sent with every request.
func WithUserAgent(ua string) Option {
	return func(c *Client) error {
		if strings.TrimSpace(ua) == "" {
			return errors.New("mpesa: user agent must not be empty")
		}
		c.userAgent = ua
		return nil
	}
}

// Environment returns the environment the client targets.
func (c *Client) Environment() Env { return c.env }

// BaseURL returns the API base URL the client talks to.
func (c *Client) BaseURL() string { return c.baseURL }

// ShortCode returns the default short code configured on the client.
func (c *Client) ShortCode() string { return c.shortCode }

// send performs an authenticated JSON request and returns the status code and
// the raw response body.
//
// A 401 means the cached token was rejected: the token is refreshed once and
// the request is replayed, which is safe because the first attempt was not
// processed.
func (c *Client) send(ctx context.Context, method, path string, payload any) (int, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, fmt.Errorf("mpesa: encode request: %w", err)
		}
	}

	endpoint := c.baseURL + path
	token, err := c.AccessToken(ctx)
	if err != nil {
		return 0, nil, err
	}

	status, raw, err := c.execute(ctx, method, endpoint, body, c.authHeaders(token))
	if err == nil {
		return status, raw, nil
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		return status, raw, err
	}

	c.invalidateToken(token)
	token, err = c.AccessToken(ctx)
	if err != nil {
		return 0, nil, err
	}
	return c.execute(ctx, method, endpoint, body, c.authHeaders(token))
}

// authHeaders returns the headers for an authenticated request.
func (c *Client) authHeaders(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
		"Accept":        "application/json",
	}
}

// execute performs a single HTTP request, retrying retryable failures.
func (c *Client) execute(ctx context.Context, method, endpoint string, body []byte, headers map[string]string) (int, []byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := sleepContext(ctx, c.backoff(attempt)); err != nil {
				return 0, nil, err
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader(body))
		if err != nil {
			return 0, nil, fmt.Errorf("mpesa: build request: %w", err)
		}
		req.Header.Set("User-Agent", c.userAgent)
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return 0, nil, ctxErr
			}
			// The request may or may not have reached Safaricom, so it is not
			// retried automatically.
			return 0, nil, fmt.Errorf("mpesa: request to %s failed: %w", endpoint, err)
		}

		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		resp.Body.Close()
		if readErr != nil {
			return 0, nil, fmt.Errorf("mpesa: read response: %w", readErr)
		}

		if resp.StatusCode < http.StatusBadRequest {
			return resp.StatusCode, raw, nil
		}

		apiErr := newAPIError(resp.StatusCode, raw)
		if !apiErr.Retryable() || attempt == c.maxRetries {
			return resp.StatusCode, raw, apiErr
		}
		lastErr = apiErr
	}

	if lastErr == nil {
		lastErr = errors.New("mpesa: request failed")
	}
	return 0, nil, lastErr
}

// doJSON posts payload to path and decodes the response into out.
func (c *Client) doJSON(ctx context.Context, path string, payload, out any) error {
	status, raw, err := c.send(ctx, http.MethodPost, path, payload)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return ErrEmptyResponse
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("mpesa: decode response from %s: %w", path, err)
	}
	if setter, ok := out.(metaSetter); ok {
		setter.setMeta(Meta{StatusCode: status, Raw: raw})
	}
	return nil
}

// Meta holds transport level details for every API response.
type Meta struct {
	// StatusCode is the HTTP status code returned by Daraja.
	StatusCode int `json:"-"`
	// Raw is the raw response body, useful for fields the SDK does not model.
	Raw []byte `json:"-"`
}

func (m *Meta) setMeta(other Meta) { *m = other }

// metaSetter is implemented by every response type through the embedded Meta.
type metaSetter interface {
	setMeta(Meta)
}

func bodyReader(b []byte) io.Reader {
	if len(b) == 0 {
		return nil
	}
	return bytes.NewReader(b)
}

// backoff returns a jittered exponential delay for the given attempt.
func (c *Client) backoff(attempt int) time.Duration {
	d := c.retryBackoff
	for i := 1; i < attempt && d < maxRetryBackoff; i++ {
		d *= 2
	}
	if d > maxRetryBackoff {
		d = maxRetryBackoff
	}
	if d <= 0 {
		return 0
	}
	half := d / 2
	if half <= 0 {
		return d
	}
	return half + time.Duration(rand.Int63n(int64(half)+1))
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// basicAuth encodes credentials for the OAuth endpoint.
func basicAuth(key, secret string) string {
	return base64.StdEncoding.EncodeToString([]byte(key + ":" + secret))
}
