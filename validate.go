package mpesa

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// validate.go holds the shared request validation helpers. Every request type
// exposes a Validate method that returns a *ValidationError describing the
// first invalid field.

func requireField(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return &ValidationError{Field: field, Reason: "is required"}
	}
	return nil
}

func requireAmount(field string, amount int64) error {
	if amount <= 0 {
		return &ValidationError{Field: field, Reason: "must be greater than zero"}
	}
	return nil
}

func requireURL(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return &ValidationError{Field: field, Reason: "is required"}
	}
	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return &ValidationError{Field: field, Reason: fmt.Sprintf("%q is not an absolute URL", value)}
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return nil
	default:
		return &ValidationError{Field: field, Reason: "must use the http or https scheme"}
	}
}

func requireOneOf(field, value string, allowed ...string) error {
	if strings.TrimSpace(value) == "" {
		return &ValidationError{Field: field, Reason: "is required"}
	}
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return &ValidationError{Field: field, Reason: fmt.Sprintf("%q must be one of: %s", value, strings.Join(allowed, ", "))}
}

// requirePhone validates an MSISDN that has already been normalised.
func requirePhone(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return &ValidationError{Field: field, Reason: "is required"}
	}
	if _, err := NormalizePhoneNumber(value); err != nil {
		if v, ok := err.(*ValidationError); ok {
			return &ValidationError{Field: field, Reason: v.Reason}
		}
		return &ValidationError{Field: field, Reason: err.Error()}
	}
	return nil
}

func requireTimestamp(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return &ValidationError{Field: field, Reason: "is required"}
	}
	if _, err := time.ParseInLocation(TimestampLayout, value, EAT()); err != nil {
		return &ValidationError{Field: field, Reason: fmt.Sprintf("%q must be in the format YYYYMMDDHHmmss", value)}
	}
	return nil
}

// withSecurityCredential returns credential when set, otherwise it generates
// one from the configured Safaricom certificate and initiator password.
func (c *Client) withSecurityCredential(credential string) (string, error) {
	if strings.TrimSpace(credential) != "" {
		return credential, nil
	}
	if c.certificate == nil || c.initiatorPwd == "" {
		return "", nil
	}
	return EncryptSecurityCredential(c.certificate, c.initiatorPwd)
}
