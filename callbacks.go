package mpesa

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// This file decodes the callbacks Safaricom posts to the URLs registered with
// the API. Callbacks are not authenticated by Safaricom, so handlers should
// verify the payment against their own records (amount, reference, receipt)
// before releasing goods or services.

// CallbackAck is the acknowledgement Daraja expects from callback endpoints.
// Write it with a 200 status code.
type CallbackAck struct {
	ResultCode int    `json:"ResultCode"`
	ResultDesc string `json:"ResultDesc"`
}

// SuccessAck acknowledges a callback.
func SuccessAck() CallbackAck {
	return CallbackAck{ResultCode: 0, ResultDesc: "Success"}
}

// FailureAck rejects a callback. Returning it from a C2B validation URL
// cancels the payment.
func FailureAck(reason string) CallbackAck {
	if reason == "" {
		reason = "Rejected"
	}
	return CallbackAck{ResultCode: 1, ResultDesc: reason}
}

// CallbackItem is a name/value pair inside the STK callback metadata.
type CallbackItem struct {
	Name  string          `json:"Name"`
	Value json.RawMessage `json:"Value"`
}

// String returns the value as a string, unquoting JSON strings.
func (i CallbackItem) String() string { return rawString(i.Value) }

// Int64 returns the value as an integer, or 0 when it is not numeric.
func (i CallbackItem) Int64() int64 { return rawInt64(i.Value) }

// Time parses the value as a Daraja YYYYMMDDHHMMSS timestamp (EAT), returning
// the zero time when it cannot be parsed.
func (i CallbackItem) Time() time.Time { return rawTime(i.Value) }

// STKCallbackMetadata is the metadata Safaricom includes for a successful STK
// push.
type STKCallbackMetadata struct {
	// Items is every metadata item, in the order Safaricom sent it.
	Items []CallbackItem `json:"Item"`

	// The fields below are extracted from Items for convenience.
	Amount             int64
	MpesaReceiptNumber string
	Balance            string
	TransactionDate    time.Time
	PhoneNumber        string
}

func (m *STKCallbackMetadata) extract() {
	for _, item := range m.Items {
		switch item.Name {
		case "Amount":
			m.Amount = item.Int64()
		case "MpesaReceiptNumber":
			m.MpesaReceiptNumber = item.String()
		case "Balance":
			m.Balance = item.String()
		case "TransactionDate":
			m.TransactionDate = item.Time()
		case "PhoneNumber":
			m.PhoneNumber = item.String()
		}
	}
}

// STKCallback is the callback Safaricom posts when a customer responds to an
// STK push prompt. Metadata is only present when ResultCode is 0.
type STKCallback struct {
	MerchantRequestID string               `json:"MerchantRequestID"`
	CheckoutRequestID string               `json:"CheckoutRequestID"`
	ResultCode        int                  `json:"ResultCode"`
	ResultDesc        string               `json:"ResultDesc"`
	Metadata          *STKCallbackMetadata `json:"CallbackMetadata,omitempty"`
}

// Succeeded reports whether the customer completed the payment.
func (c *STKCallback) Succeeded() bool { return c.ResultCode == 0 }

// Cancelled reports whether the customer cancelled the prompt.
func (c *STKCallback) Cancelled() bool { return c.ResultCode == 1032 }

// TimedOut reports whether the prompt expired before the customer responded.
func (c *STKCallback) TimedOut() bool { return c.ResultCode == 1037 }

// Receipt returns the M-Pesa receipt number, or an empty string when the
// payment did not succeed.
func (c *STKCallback) Receipt() string {
	if c.Metadata == nil {
		return ""
	}
	return c.Metadata.MpesaReceiptNumber
}

// ParseSTKCallback decodes the JSON body Safaricom posts to an STK push
// callback URL.
func ParseSTKCallback(data []byte) (*STKCallback, error) {
	var envelope struct {
		Body *struct {
			STKCallback *STKCallback `json:"stkCallback"`
		} `json:"Body"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("mpesa: parse STK callback: %w", err)
	}

	var callback *STKCallback
	if envelope.Body != nil {
		callback = envelope.Body.STKCallback
	}
	if callback == nil {
		// Some gateways forward the inner object without the Body wrapper.
		var direct STKCallback
		if err := json.Unmarshal(data, &direct); err != nil {
			return nil, fmt.Errorf("mpesa: parse STK callback: %w", err)
		}
		if direct.CheckoutRequestID == "" && direct.MerchantRequestID == "" && direct.ResultDesc == "" {
			return nil, errors.New("mpesa: payload is not an STK callback")
		}
		callback = &direct
	}
	if callback.Metadata != nil {
		callback.Metadata.extract()
	}
	return callback, nil
}

// KeyValue is a name/value pair in an asynchronous result callback.
type KeyValue struct {
	Key   string          `json:"Key"`
	Value json.RawMessage `json:"Value"`
}

// String returns the value as a string, unquoting JSON strings.
func (kv KeyValue) String() string { return rawString(kv.Value) }

// Int64 returns the value as an integer, or 0 when it is not numeric.
func (kv KeyValue) Int64() int64 { return rawInt64(kv.Value) }

// Float64 returns the value as a float, or 0 when it is not numeric.
func (kv KeyValue) Float64() float64 {
	v, _ := strconv.ParseFloat(strings.Trim(rawString(kv.Value), `"`), 64)
	return v
}

// KeyValueList accepts either a single object or an array of objects, because
// Daraja is inconsistent about ReferenceData.
type KeyValueList []KeyValue

// UnmarshalJSON implements json.Unmarshaler.
func (l *KeyValueList) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	switch {
	case trimmed == "" || trimmed == "null":
		*l = nil
		return nil
	case strings.HasPrefix(trimmed, "["):
		var items []KeyValue
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		*l = items
		return nil
	default:
		var item KeyValue
		if err := json.Unmarshal(data, &item); err != nil {
			return err
		}
		*l = KeyValueList{item}
		return nil
	}
}

// ResultParameters holds the parameters of an asynchronous result.
type ResultParameters struct {
	ResultParameter []KeyValue `json:"ResultParameter"`
}

// ReferenceData holds the reference items of an asynchronous result.
type ReferenceData struct {
	ReferenceItem KeyValueList `json:"ReferenceItem"`
}

// ResultCallback is the asynchronous result posted for B2C, B2B, reversal,
// account balance and transaction status requests. Queue timeout notifications
// use the same shape with ResultType 1.
type ResultCallback struct {
	ResultType               int               `json:"ResultType"`
	ResultCode               int               `json:"ResultCode"`
	ResultDesc               string            `json:"ResultDesc"`
	OriginatorConversationID string            `json:"OriginatorConversationID"`
	ConversationID           string            `json:"ConversationID"`
	TransactionID            string            `json:"TransactionID"`
	ResultParameters         *ResultParameters `json:"ResultParameters,omitempty"`
	ReferenceData            *ReferenceData    `json:"ReferenceData,omitempty"`
}

// Succeeded reports whether the request completed successfully.
func (r *ResultCallback) Succeeded() bool { return r.ResultCode == 0 }

// TimedOut reports whether the callback is a queue timeout notification.
func (r *ResultCallback) TimedOut() bool { return r.ResultType == 1 }

// Param returns the result parameter with the given key.
func (r *ResultCallback) Param(key string) (string, bool) {
	if r == nil {
		return "", false
	}
	if r.ResultParameters != nil {
		for _, param := range r.ResultParameters.ResultParameter {
			if param.Key == key {
				return param.String(), true
			}
		}
	}
	if r.ReferenceData != nil {
		for _, item := range r.ReferenceData.ReferenceItem {
			if item.Key == key {
				return item.String(), true
			}
		}
	}
	return "", false
}

// ParamFloat returns the numeric result parameter with the given key.
func (r *ResultCallback) ParamFloat(key string) (float64, bool) {
	if r == nil {
		return 0, false
	}
	if r.ResultParameters != nil {
		for _, param := range r.ResultParameters.ResultParameter {
			if param.Key == key {
				return param.Float64(), true
			}
		}
	}
	return 0, false
}

// ParseResultCallback decodes the JSON body Safaricom posts to the ResultURL or
// QueueTimeOutURL of an asynchronous request.
func ParseResultCallback(data []byte) (*ResultCallback, error) {
	var envelope struct {
		Result *ResultCallback `json:"Result"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("mpesa: parse result callback: %w", err)
	}
	if envelope.Result != nil {
		if envelope.Result.ResultDesc == "" && envelope.Result.ConversationID == "" && envelope.Result.TransactionID == "" {
			return nil, errors.New("mpesa: payload is not a result callback")
		}
		return envelope.Result, nil
	}

	var direct ResultCallback
	if err := json.Unmarshal(data, &direct); err != nil {
		return nil, fmt.Errorf("mpesa: parse result callback: %w", err)
	}
	if direct.ResultDesc == "" && direct.ConversationID == "" && direct.TransactionID == "" {
		return nil, errors.New("mpesa: payload is not a result callback")
	}
	return &direct, nil
}

// C2BConfirmation is the callback Safaricom posts when a customer pays a pay
// bill or till number.
type C2BConfirmation struct {
	TransactionType   string
	TransID           string
	TransTime         string
	TransAmount       string
	BusinessShortCode string
	BillRefNumber     string
	InvoiceNumber     string
	OrgAccountBalance string
	ThirdPartyTransID string
	MSISDN            string
	FirstName         string
	MiddleName        string
	LastName          string

	// Raw is the raw callback body.
	Raw []byte
}

// UnmarshalJSON implements json.Unmarshaler, accepting numeric or string values
// for the amount, balance and MSISDN fields, which Daraja is inconsistent
// about.
func (c *C2BConfirmation) UnmarshalJSON(data []byte) error {
	var payload struct {
		TransactionType   flexString `json:"TransactionType"`
		TransID           flexString `json:"TransID"`
		TransTime         flexString `json:"TransTime"`
		TransAmount       flexString `json:"TransAmount"`
		BusinessShortCode flexString `json:"BusinessShortCode"`
		BillRefNumber     flexString `json:"BillRefNumber"`
		InvoiceNumber     flexString `json:"InvoiceNumber"`
		OrgAccountBalance flexString `json:"OrgAccountBalance"`
		ThirdPartyTransID flexString `json:"ThirdPartyTransID"`
		MSISDN            flexString `json:"MSISDN"`
		FirstName         flexString `json:"FirstName"`
		MiddleName        flexString `json:"MiddleName"`
		LastName          flexString `json:"LastName"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	c.TransactionType = string(payload.TransactionType)
	c.TransID = string(payload.TransID)
	c.TransTime = string(payload.TransTime)
	c.TransAmount = string(payload.TransAmount)
	c.BusinessShortCode = string(payload.BusinessShortCode)
	c.BillRefNumber = string(payload.BillRefNumber)
	c.InvoiceNumber = string(payload.InvoiceNumber)
	c.OrgAccountBalance = string(payload.OrgAccountBalance)
	c.ThirdPartyTransID = string(payload.ThirdPartyTransID)
	c.MSISDN = string(payload.MSISDN)
	c.FirstName = string(payload.FirstName)
	c.MiddleName = string(payload.MiddleName)
	c.LastName = string(payload.LastName)
	c.Raw = append(c.Raw[:0], data...)
	return nil
}

// Amount returns TransAmount as a number.
func (c *C2BConfirmation) Amount() (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(c.TransAmount), 64)
	if err != nil {
		return 0, fmt.Errorf("mpesa: parse C2B amount %q: %w", c.TransAmount, err)
	}
	return v, nil
}

// CustomerName returns the payer name as sent by Safaricom.
func (c *C2BConfirmation) CustomerName() string {
	parts := make([]string, 0, 3)
	for _, part := range []string{c.FirstName, c.MiddleName, c.LastName} {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, " ")
}

// ParseC2BConfirmation decodes the JSON body Safaricom posts to a C2B
// confirmation URL.
func ParseC2BConfirmation(data []byte) (*C2BConfirmation, error) {
	var confirmation C2BConfirmation
	if err := json.Unmarshal(data, &confirmation); err != nil {
		return nil, fmt.Errorf("mpesa: parse C2B confirmation: %w", err)
	}
	if confirmation.TransID == "" && confirmation.MSISDN == "" && confirmation.BillRefNumber == "" {
		return nil, errors.New("mpesa: payload is not a C2B confirmation")
	}
	return &confirmation, nil
}

// flexString accepts both JSON strings and JSON numbers.
type flexString string

// UnmarshalJSON implements json.Unmarshaler.
func (f *flexString) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*f = ""
		return nil
	}
	if strings.HasPrefix(trimmed, `"`) {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*f = flexString(s)
		return nil
	}
	*f = flexString(trimmed)
	return nil
}

// rawString renders a json.RawMessage as a plain string.
func rawString(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	if strings.HasPrefix(trimmed, `"`) {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return s
		}
	}
	return trimmed
}

// rawInt64 parses a json.RawMessage as an integer.
func rawInt64(raw json.RawMessage) int64 {
	trimmed := strings.Trim(rawString(raw), `"`)
	v, err := strconv.ParseFloat(strings.TrimSpace(trimmed), 64)
	if err != nil {
		return 0
	}
	return int64(v)
}

// rawTime parses a Daraja timestamp such as 20170824155055.
func rawTime(raw json.RawMessage) time.Time {
	s := strings.TrimSpace(strings.Trim(rawString(raw), `"`))
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		s = strconv.FormatInt(int64(v), 10)
	}
	if t, err := time.ParseInLocation(TimestampLayout, s, eat); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}
