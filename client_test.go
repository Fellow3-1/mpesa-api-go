package mpesa

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testAppKey      = "test-app-key"
	testAppSecret   = "test-app-secret"
	testPassKey     = "bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada1ed2c919"
	testShortCode   = "174379"
	testCallbackURL = "https://example.com/mpesa/callback"
)

// ---------------------------------------------------------------------------
// Test harness
// ---------------------------------------------------------------------------

type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   []byte
}

type recorder struct {
	mu       sync.Mutex
	requests []recordedRequest
}

func (r *recorder) add(req recordedRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req)
}

func (r *recorder) all() []recordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]recordedRequest, len(r.requests))
	copy(out, r.requests)
	return out
}

// count returns how many requests were made to path.
func (r *recorder) count(path string) int {
	n := 0
	for _, req := range r.all() {
		if req.Path == path {
			n++
		}
	}
	return n
}

func (r *recorder) last() recordedRequest {
	reqs := r.all()
	if len(reqs) == 0 {
		return recordedRequest{}
	}
	return reqs[len(reqs)-1]
}

type testHandler func(w http.ResponseWriter, r *http.Request, body []byte)

func newTestClientWithOptions(t *testing.T, handler testHandler, opts ...Option) (*Client, *recorder) {
	t.Helper()

	rec := &recorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		rec.add(recordedRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Query:  r.URL.RawQuery,
			Header: r.Header.Clone(),
			Body:   body,
		})
		handler(w, r, body)
	}))
	t.Cleanup(server.Close)

	base := []Option{
		WithBaseURL(server.URL),
		WithShortCode(testShortCode),
		WithPassKey(testPassKey),
	}
	client, err := New(testAppKey, testAppSecret, DEV, append(base, opts...)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client, rec
}

func newTestClient(t *testing.T, handler testHandler) (*Client, *recorder) {
	t.Helper()
	return newTestClientWithOptions(t, handler)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeToken(w http.ResponseWriter, token, expiresIn string) {
	writeJSON(w, http.StatusOK, map[string]string{
		"access_token": token,
		"expires_in":   expiresIn,
	})
}

// withOAuth serves the OAuth endpoint and delegates every other request to api.
func withOAuth(token string, api testHandler) testHandler {
	return func(w http.ResponseWriter, r *http.Request, body []byte) {
		if r.URL.Path == "/oauth/v1/generate" {
			writeToken(w, token, "3599")
			return
		}
		api(w, r, body)
	}
}

func decodeBody(t *testing.T, body []byte) map[string]any {
	t.Helper()
	if len(body) == 0 {
		t.Fatal("request body was empty")
	}
	if bytes.HasPrefix(bytes.TrimSpace(body), []byte("[")) {
		t.Fatalf("request body must be a JSON object, got an array: %s", body)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode request body %s: %v", body, err)
	}
	return decoded
}

func assertString(t *testing.T, payload map[string]any, key, want string) {
	t.Helper()
	got, ok := payload[key]
	if !ok {
		t.Errorf("request body is missing %q: %v", key, payload)
		return
	}
	if got != want {
		t.Errorf("request body %s = %v, want %q", key, got, want)
	}
}

func assertNumber(t *testing.T, payload map[string]any, key string, want float64) {
	t.Helper()
	got, ok := payload[key]
	if !ok {
		t.Errorf("request body is missing %q: %v", key, payload)
		return
	}
	if got != want {
		t.Errorf("request body %s = %v, want %v", key, got, want)
	}
}

func assertNoKey(t *testing.T, payload map[string]any, key string) {
	t.Helper()
	if _, ok := payload[key]; ok {
		t.Errorf("request body unexpectedly contains %q", key)
	}
}

func stkSuccessPayload() map[string]any {
	return map[string]any{
		"MerchantRequestID":   "29115-34620561-1",
		"CheckoutRequestID":   "ws_CO_191220191020363925",
		"ResponseCode":        "0",
		"ResponseDescription": "Success. Request accepted for processing",
		"CustomerMessage":     "Success. Request accepted for processing",
	}
}

func validSTKPushRequest() STKPushRequest {
	return STKPushRequest{
		Amount:           100,
		PhoneNumber:      "0712345678",
		CallBackURL:      testCallbackURL,
		AccountReference: "INV-001",
		TransactionDesc:  "Invoice 001",
	}
}

// ---------------------------------------------------------------------------
// Construction
// ---------------------------------------------------------------------------

func TestNewRequiresCredentials(t *testing.T) {
	if _, err := New("", "secret", DEV); !errors.Is(err, ErrMissingCredentials) {
		t.Fatalf("New with empty key: got %v, want ErrMissingCredentials", err)
	}
	if _, err := New("key", "   ", DEV); !errors.Is(err, ErrMissingCredentials) {
		t.Fatalf("New with blank secret: got %v, want ErrMissingCredentials", err)
	}
}

func TestNewRejectsUnknownEnvironment(t *testing.T) {
	if _, err := New("key", "secret", Env("staging")); !errors.Is(err, ErrInvalidEnvironment) {
		t.Fatalf("New with unknown env: got %v, want ErrInvalidEnvironment", err)
	}
}

func TestNewDefaults(t *testing.T) {
	client, err := New("key", "secret", "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.Environment() != DEV {
		t.Errorf("Environment() = %q, want %q", client.Environment(), DEV)
	}
	if client.BaseURL() != SandboxBaseURL {
		t.Errorf("BaseURL() = %q, want %q", client.BaseURL(), SandboxBaseURL)
	}

	prod, err := New("key", "secret", PRODUCTION)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if prod.BaseURL() != ProductionBaseURL {
		t.Errorf("BaseURL() = %q, want %q", prod.BaseURL(), ProductionBaseURL)
	}
}

func TestOptionValidation(t *testing.T) {
	tests := []struct {
		name string
		opt  Option
	}{
		{"relative base URL", WithBaseURL("/relative")},
		{"empty base URL", WithBaseURL("")},
		{"unsupported scheme", WithBaseURL("ftp://example.com")},
		{"zero timeout", WithTimeout(0)},
		{"negative retries", WithRetries(-1)},
		{"negative backoff", WithRetryBackoff(-time.Second)},
		{"blank user agent", WithUserAgent(" ")},
		{"nil http client", WithHTTPClient(nil)},
		{"nil certificate", WithCertificate(nil)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New("key", "secret", DEV, tc.opt); err == nil {
				t.Fatalf("New with %s: expected an error", tc.name)
			}
		})
	}
}

func TestWithBaseURLTrimsTrailingSlash(t *testing.T) {
	client, err := New("key", "secret", DEV, WithBaseURL("https://example.com/"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.BaseURL() != "https://example.com" {
		t.Errorf("BaseURL() = %q", client.BaseURL())
	}
}

// ---------------------------------------------------------------------------
// Authentication
// ---------------------------------------------------------------------------

func TestAccessTokenIsCached(t *testing.T) {
	var oauthCalls int32
	client, rec := newTestClient(t, func(w http.ResponseWriter, r *http.Request, _ []byte) {
		if r.URL.Path != "/oauth/v1/generate" {
			http.NotFound(w, r)
			return
		}
		atomic.AddInt32(&oauthCalls, 1)
		writeToken(w, "token-1", "3599")
	})

	for i := 0; i < 3; i++ {
		token, err := client.AccessToken(context.Background())
		if err != nil {
			t.Fatalf("AccessToken: %v", err)
		}
		if token != "token-1" {
			t.Fatalf("token = %q, want token-1", token)
		}
	}
	if oauthCalls != 1 {
		t.Errorf("oauth endpoint called %d times, want 1", oauthCalls)
	}

	req := rec.all()[0]
	if req.Method != http.MethodGet {
		t.Errorf("oauth method = %s, want GET", req.Method)
	}
	if req.Query != "grant_type=client_credentials" {
		t.Errorf("oauth query = %q", req.Query)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(testAppKey+":"+testAppSecret))
	if got := req.Header.Get("Authorization"); got != wantAuth {
		t.Errorf("oauth Authorization = %q, want %q", got, wantAuth)
	}
	if len(req.Body) != 0 {
		t.Errorf("oauth request must not have a body, got %q", req.Body)
	}
}

func TestAccessTokenRefreshesWhenExpired(t *testing.T) {
	var oauthCalls int32
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request, _ []byte) {
		n := atomic.AddInt32(&oauthCalls, 1)
		writeToken(w, fmt.Sprintf("token-%d", n), "3599")
	})

	now := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	client.now = func() time.Time { return now }

	first, err := client.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}

	// Ten minutes later the token is still valid (lifetime is 3599s).
	now = now.Add(10 * time.Minute)
	stillValid, err := client.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if stillValid != first {
		t.Errorf("token changed before expiry: %q -> %q", first, stillValid)
	}
	if oauthCalls != 1 {
		t.Fatalf("oauth called %d times, want 1", oauthCalls)
	}

	// Past the expiry (minus the safety skew) a new token is fetched.
	now = now.Add(60 * time.Minute)
	refreshed, err := client.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if refreshed != "token-2" {
		t.Errorf("token = %q, want token-2", refreshed)
	}
	if oauthCalls != 2 {
		t.Errorf("oauth called %d times, want 2", oauthCalls)
	}
}

func TestAccessTokenAcceptsNumericExpiresIn(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, map[string]any{"access_token": "token", "expires_in": 3599})
	})
	if _, err := client.AccessToken(context.Background()); err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
}

func TestAccessTokenRejectsEmptyToken(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, map[string]any{"expires_in": "3599"})
	})
	if _, err := client.AccessToken(context.Background()); err == nil {
		t.Fatal("expected an error for a response without an access token")
	}
}

func TestAuthenticationFailureIsWrapped(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"requestId":    "req-1",
			"errorCode":    "401.002.01",
			"errorMessage": "Error Occurred - Invalid Access Token",
		})
	})

	_, err := client.AccessToken(context.Background())
	if err == nil {
		t.Fatal("expected an authentication error")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("error = %v, want it to mention authentication", err)
	}
	apiErr, ok := AsAPIError(err)
	if !ok || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("error = %v, want a 401 APIError", err)
	}
}

func TestUnauthorizedRefreshesTokenAndReplays(t *testing.T) {
	var oauthCalls int32
	client, rec := newTestClient(t, func(w http.ResponseWriter, r *http.Request, _ []byte) {
		if r.URL.Path == "/oauth/v1/generate" {
			n := atomic.AddInt32(&oauthCalls, 1)
			writeToken(w, fmt.Sprintf("token-%d", n), "3599")
			return
		}
		if r.Header.Get("Authorization") == "Bearer token-1" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"requestId":    "req-1",
				"errorCode":    "401.002.01",
				"errorMessage": "Invalid Access Token",
			})
			return
		}
		writeJSON(w, http.StatusOK, stkSuccessPayload())
	})

	res, err := client.STKPush(context.Background(), validSTKPushRequest())
	if err != nil {
		t.Fatalf("STKPush: %v", err)
	}
	if !res.Accepted() {
		t.Errorf("response was not accepted: %+v", res)
	}
	if oauthCalls != 2 {
		t.Errorf("oauth called %d times, want 2", oauthCalls)
	}
	if got := rec.count("/mpesa/stkpush/v1/processrequest"); got != 2 {
		t.Errorf("STK push called %d times, want 2", got)
	}
}

// ---------------------------------------------------------------------------
// API errors and retries
// ---------------------------------------------------------------------------

func TestAPIErrorIsParsed(t *testing.T) {
	client, _ := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"requestId":    "req-1",
			"errorCode":    "400.002.02",
			"errorMessage": "Bad Request - Invalid Amount",
		})
	}))

	_, err := client.STKPush(context.Background(), validSTKPushRequest())
	if err == nil {
		t.Fatal("expected an API error")
	}
	apiErr, ok := AsAPIError(err)
	if !ok {
		t.Fatalf("error %v is not an *APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d", apiErr.StatusCode)
	}
	if apiErr.ErrorCode != "400.002.02" {
		t.Errorf("ErrorCode = %q", apiErr.ErrorCode)
	}
	if apiErr.RequestID != "req-1" {
		t.Errorf("RequestID = %q", apiErr.RequestID)
	}
	msg := apiErr.Error()
	for _, want := range []string{"400.002.02", "Invalid Amount", "req-1"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message %q does not contain %q", msg, want)
		}
	}
	if apiErr.Retryable() {
		t.Error("a 400 response must not be retryable")
	}
}

func TestAPIErrorWithNonJSONBody(t *testing.T) {
	client, _ := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>bad gateway</html>"))
	}))

	_, err := client.STKPush(context.Background(), validSTKPushRequest())
	apiErr, ok := AsAPIError(err)
	if !ok {
		t.Fatalf("error %v is not an *APIError", err)
	}
	if apiErr.ErrorMessage != http.StatusText(http.StatusBadGateway) {
		t.Errorf("ErrorMessage = %q", apiErr.ErrorMessage)
	}
}

func TestRetryableStatusIsRetried(t *testing.T) {
	var apiCalls int32
	client, _ := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		if atomic.AddInt32(&apiCalls, 1) <= 2 {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"errorMessage": "Service Unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, stkSuccessPayload())
	}))
	client.maxRetries = 2
	client.retryBackoff = 0

	if _, err := client.STKPush(context.Background(), validSTKPushRequest()); err != nil {
		t.Fatalf("STKPush: %v", err)
	}
	if apiCalls != 3 {
		t.Errorf("api called %d times, want 3", apiCalls)
	}
}

func TestServerErrorsAreNotRetried(t *testing.T) {
	var apiCalls int32
	client, _ := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		atomic.AddInt32(&apiCalls, 1)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"errorMessage": "Internal Server Error"})
	}))
	// Even with retries configured, a 500 may have been processed by Safaricom,
	// so it must not be replayed.
	client.maxRetries = 2
	client.retryBackoff = 0

	if _, err := client.STKPush(context.Background(), validSTKPushRequest()); err == nil {
		t.Fatal("expected an error")
	}
	if apiCalls != 1 {
		t.Errorf("api called %d times, want 1", apiCalls)
	}
}

func TestContextCancellation(t *testing.T) {
	client, _ := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		time.Sleep(200 * time.Millisecond)
		writeJSON(w, http.StatusOK, stkSuccessPayload())
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := client.STKPush(ctx, validSTKPushRequest())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
}

func TestEmptyResponseBody(t *testing.T) {
	client, _ := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		w.WriteHeader(http.StatusOK)
	}))
	if _, err := client.STKPush(context.Background(), validSTKPushRequest()); !errors.Is(err, ErrEmptyResponse) {
		t.Fatalf("error = %v, want ErrEmptyResponse", err)
	}
}
