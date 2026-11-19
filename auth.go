package mpesa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// defaultTokenTTL is used when the OAuth response does not say how long the
	// token is valid. Safaricom issues tokens for 3599 seconds.
	defaultTokenTTL = 3599 * time.Second

	// tokenExpirySkew refreshes the token slightly before it actually expires
	// so that a token cannot expire mid request.
	tokenExpirySkew = 60 * time.Second

	// minTokenTTL prevents a tight refresh loop if Daraja reports a very short
	// lifetime.
	minTokenTTL = 30 * time.Second
)

// tokenResponse is the payload returned by the OAuth endpoint.
//
// Daraja returns expires_in as a JSON string ("3599"), so it is decoded through
// flexInt.
type tokenResponse struct {
	AccessToken string  `json:"access_token"`
	ExpiresIn   flexInt `json:"expires_in"`
}

// flexInt accepts both JSON numbers and numeric strings.
type flexInt int64

// UnmarshalJSON implements json.Unmarshaler.
func (f *flexInt) UnmarshalJSON(data []byte) error {
	s := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("mpesa: cannot parse %q as a number", s)
	}
	*f = flexInt(v)
	return nil
}

// AccessToken returns a valid OAuth access token, fetching and caching a new one
// when the cached token is missing or about to expire.
func (c *Client) AccessToken(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.token != "" && c.now().Before(c.tokenExpiry) {
		return c.token, nil
	}

	token, ttl, err := c.fetchToken(ctx)
	if err != nil {
		return "", err
	}

	c.token = token
	c.tokenExpiry = c.now().Add(ttl)
	return token, nil
}

// fetchToken requests a new token from the OAuth endpoint.
func (c *Client) fetchToken(ctx context.Context) (string, time.Duration, error) {
	headers := map[string]string{
		"Authorization": "Basic " + basicAuth(c.appKey, c.appSecret),
		"Accept":        "application/json",
	}

	status, raw, err := c.execute(ctx, http.MethodGet, c.baseURL+pathOAuth, nil, headers)
	if err != nil {
		if apiErr, ok := AsAPIError(err); ok && isCredentialError(apiErr.StatusCode) {
			return "", 0, fmt.Errorf("mpesa: authentication failed, check the app key and app secret: %w", err)
		}
		return "", 0, err
	}

	var payload tokenResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", 0, fmt.Errorf("mpesa: decode token response (http %d): %w", status, err)
	}
	if payload.AccessToken == "" {
		return "", 0, fmt.Errorf("mpesa: token response from %s did not contain an access token", c.baseURL)
	}

	ttl := time.Duration(payload.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	ttl -= tokenExpirySkew
	if ttl < minTokenTTL {
		ttl = minTokenTTL
	}
	return payload.AccessToken, ttl, nil
}

// isCredentialError reports whether a token endpoint status code means the app
// key or app secret was rejected.
func isCredentialError(status int) bool {
	switch status {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return true
	}
	return false
}

// invalidateToken drops the cached token if it is still the one that was
// rejected, so that a concurrent goroutine's fresh token is not discarded.
func (c *Client) invalidateToken(token string) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.token == token {
		c.token = ""
		c.tokenExpiry = time.Time{}
	}
}
