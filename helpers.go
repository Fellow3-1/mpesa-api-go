package mpesa

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// TimestampLayout is the timestamp format Daraja expects: YYYYMMDDHHmmss in
// East Africa Time.
const TimestampLayout = "20060102150405"

// eat is East Africa Time, the time zone Safaricom uses for timestamps.
var eat = time.FixedZone("EAT", 3*60*60)

// EAT returns the East Africa Time location used for Daraja timestamps.
func EAT() *time.Location { return eat }

// FormatTimestamp formats t the way Daraja expects (YYYYMMDDHHmmss in EAT).
func FormatTimestamp(t time.Time) string {
	return t.In(eat).Format(TimestampLayout)
}

// GenerateSTKPushPassword builds the STK push password, which is the base64
// encoding of shortCode + passKey + timestamp.
func GenerateSTKPushPassword(shortCode, passKey, timestamp string) string {
	return base64.StdEncoding.EncodeToString([]byte(shortCode + passKey + timestamp))
}

// NormalizePhoneNumber converts a Kenyan phone number to the 2547XXXXXXXX form
// Daraja expects. It accepts 07XXXXXXXX, 01XXXXXXXX, 7XXXXXXXX, 2547XXXXXXXX,
// +2547XXXXXXXX and the same numbers with spaces, dashes or parentheses.
func NormalizePhoneNumber(phone string) (string, error) {
	original := phone

	s := strings.TrimSpace(phone)
	s = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", "+", "").Replace(s)
	s = strings.TrimPrefix(s, "00")

	for _, r := range s {
		if r < '0' || r > '9' {
			return "", &ValidationError{Field: "phone number", Reason: fmt.Sprintf("%q is not a valid Kenyan phone number", original)}
		}
	}

	switch {
	case len(s) == 12 && strings.HasPrefix(s, "254"):
		// Already normalised.
	case len(s) == 13 && strings.HasPrefix(s, "2540"):
		// 25407XXXXXXXX -> 2547XXXXXXXX / 25401XXXXXXXX -> 2541XXXXXXXX
		s = "254" + s[4:]
	case len(s) == 10 && strings.HasPrefix(s, "0"):
		s = "254" + s[1:]
	case len(s) == 9 && (strings.HasPrefix(s, "7") || strings.HasPrefix(s, "1")):
		s = "254" + s
	default:
		return "", &ValidationError{Field: "phone number", Reason: fmt.Sprintf("%q is not a valid Kenyan phone number", original)}
	}

	subscriber := strings.TrimPrefix(s, "254")
	if len(subscriber) != 9 || (subscriber[0] != '7' && subscriber[0] != '1') {
		return "", &ValidationError{Field: "phone number", Reason: fmt.Sprintf("%q is not a valid Kenyan phone number", original)}
	}
	return s, nil
}
