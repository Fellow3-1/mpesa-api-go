package mpesa

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// The tests in this file assert that every request type rejects an incomplete
// payload before a network call is made.

func TestB2CValidate(t *testing.T) {
	base := B2CRequest{
		InitiatorName:      "testapi",
		SecurityCredential: "credential",
		CommandID:          CommandIDBusinessPayment,
		Amount:             10,
		PartyA:             "600000",
		PartyB:             "254708374149",
		Remarks:            "payout",
		QueueTimeOutURL:    testCallbackURL,
		ResultURL:          testCallbackURL,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	tests := []struct {
		name   string
		field  string
		mutate func(*B2CRequest)
	}{
		{"missing initiator", "InitiatorName", func(r *B2CRequest) { r.InitiatorName = "" }},
		{"missing credential", "SecurityCredential", func(r *B2CRequest) { r.SecurityCredential = "" }},
		{"unknown command", "CommandID", func(r *B2CRequest) { r.CommandID = "Nope" }},
		{"zero amount", "Amount", func(r *B2CRequest) { r.Amount = 0 }},
		{"missing party A", "PartyA", func(r *B2CRequest) { r.PartyA = "" }},
		{"bad party B", "PartyB", func(r *B2CRequest) { r.PartyB = "12345" }},
		{"missing remarks", "Remarks", func(r *B2CRequest) { r.Remarks = "" }},
		{"missing result url", "ResultURL", func(r *B2CRequest) { r.ResultURL = "" }},
		{"missing timeout url", "QueueTimeOutURL", func(r *B2CRequest) { r.QueueTimeOutURL = "" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base
			tc.mutate(&req)
			assertValidationField(t, req.Validate(), tc.field)
		})
	}
}

func TestB2BValidate(t *testing.T) {
	base := B2BRequest{
		InitiatorName:          "testapi",
		SecurityCredential:     "credential",
		CommandID:              CommandIDBusinessPayBill,
		SenderIdentifierType:   "4",
		ReceiverIdentifierType: "4",
		Amount:                 10,
		PartyA:                 "600000",
		PartyB:                 "600001",
		AccountReference:       "ref",
		Remarks:                "stock",
		QueueTimeOutURL:        testCallbackURL,
		ResultURL:              testCallbackURL,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	tests := []struct {
		name   string
		field  string
		mutate func(*B2BRequest)
	}{
		{"missing initiator", "InitiatorName", func(r *B2BRequest) { r.InitiatorName = "" }},
		{"missing credential", "SecurityCredential", func(r *B2BRequest) { r.SecurityCredential = "" }},
		{"unknown command", "CommandID", func(r *B2BRequest) { r.CommandID = "Nope" }},
		{"missing sender type", "SenderIdentifierType", func(r *B2BRequest) { r.SenderIdentifierType = "" }},
		{"missing receiver type", "ReceiverIdentifierType", func(r *B2BRequest) { r.ReceiverIdentifierType = "" }},
		{"zero amount", "Amount", func(r *B2BRequest) { r.Amount = 0 }},
		{"missing party A", "PartyA", func(r *B2BRequest) { r.PartyA = "" }},
		{"missing party B", "PartyB", func(r *B2BRequest) { r.PartyB = "" }},
		{"missing reference", "AccountReference", func(r *B2BRequest) { r.AccountReference = "" }},
		{"missing remarks", "Remarks", func(r *B2BRequest) { r.Remarks = "" }},
		{"missing result url", "ResultURL", func(r *B2BRequest) { r.ResultURL = "" }},
		{"missing timeout url", "QueueTimeOutURL", func(r *B2BRequest) { r.QueueTimeOutURL = "" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base
			tc.mutate(&req)
			assertValidationField(t, req.Validate(), tc.field)
		})
	}
}

func TestReversalValidate(t *testing.T) {
	base := ReversalRequest{
		InitiatorName:          "testapi",
		SecurityCredential:     "credential",
		CommandID:              CommandIDTransactionReversal,
		TransactionID:          "NLJ7RT61SV",
		Amount:                 10,
		ReceiverParty:          "600000",
		ReceiverIdentifierType: "4",
		ResultURL:              testCallbackURL,
		QueueTimeOutURL:        testCallbackURL,
		Remarks:                "wrong number",
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	tests := []struct {
		name   string
		field  string
		mutate func(*ReversalRequest)
	}{
		{"missing initiator", "InitiatorName", func(r *ReversalRequest) { r.InitiatorName = "" }},
		{"missing credential", "SecurityCredential", func(r *ReversalRequest) { r.SecurityCredential = "" }},
		{"unknown command", "CommandID", func(r *ReversalRequest) { r.CommandID = "Nope" }},
		{"missing transaction", "TransactionID", func(r *ReversalRequest) { r.TransactionID = "" }},
		{"zero amount", "Amount", func(r *ReversalRequest) { r.Amount = 0 }},
		{"missing receiver", "ReceiverParty", func(r *ReversalRequest) { r.ReceiverParty = "" }},
		{"missing receiver type", "ReceiverIdentifierType", func(r *ReversalRequest) { r.ReceiverIdentifierType = "" }},
		{"missing result url", "ResultURL", func(r *ReversalRequest) { r.ResultURL = "" }},
		{"missing timeout url", "QueueTimeOutURL", func(r *ReversalRequest) { r.QueueTimeOutURL = "" }},
		{"missing remarks", "Remarks", func(r *ReversalRequest) { r.Remarks = "" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base
			tc.mutate(&req)
			assertValidationField(t, req.Validate(), tc.field)
		})
	}
}

func TestAccountBalanceValidate(t *testing.T) {
	base := AccountBalanceRequest{
		InitiatorName:      "testapi",
		SecurityCredential: "credential",
		CommandID:          CommandIDAccountBalance,
		PartyA:             "600000",
		IdentifierType:     "4",
		Remarks:            "balance",
		QueueTimeOutURL:    testCallbackURL,
		ResultURL:          testCallbackURL,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	tests := []struct {
		name   string
		field  string
		mutate func(*AccountBalanceRequest)
	}{
		{"missing initiator", "InitiatorName", func(r *AccountBalanceRequest) { r.InitiatorName = "" }},
		{"missing credential", "SecurityCredential", func(r *AccountBalanceRequest) { r.SecurityCredential = "" }},
		{"unknown command", "CommandID", func(r *AccountBalanceRequest) { r.CommandID = "Nope" }},
		{"missing party", "PartyA", func(r *AccountBalanceRequest) { r.PartyA = "" }},
		{"missing identifier", "IdentifierType", func(r *AccountBalanceRequest) { r.IdentifierType = "" }},
		{"missing remarks", "Remarks", func(r *AccountBalanceRequest) { r.Remarks = "" }},
		{"missing timeout url", "QueueTimeOutURL", func(r *AccountBalanceRequest) { r.QueueTimeOutURL = "" }},
		{"missing result url", "ResultURL", func(r *AccountBalanceRequest) { r.ResultURL = "" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base
			tc.mutate(&req)
			assertValidationField(t, req.Validate(), tc.field)
		})
	}
}

func TestTransactionStatusValidate(t *testing.T) {
	base := TransactionStatusRequest{
		InitiatorName:      "testapi",
		SecurityCredential: "credential",
		CommandID:          CommandIDTransactionStatus,
		TransactionID:      "NLJ7RT61SV",
		PartyA:             "600000",
		IdentifierType:     "4",
		Remarks:            "status",
		ResultURL:          testCallbackURL,
		QueueTimeOutURL:    testCallbackURL,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	tests := []struct {
		name   string
		field  string
		mutate func(*TransactionStatusRequest)
	}{
		{"missing initiator", "InitiatorName", func(r *TransactionStatusRequest) { r.InitiatorName = "" }},
		{"missing credential", "SecurityCredential", func(r *TransactionStatusRequest) { r.SecurityCredential = "" }},
		{"unknown command", "CommandID", func(r *TransactionStatusRequest) { r.CommandID = "Nope" }},
		{"missing transaction", "TransactionID", func(r *TransactionStatusRequest) { r.TransactionID = "" }},
		{"missing party", "PartyA", func(r *TransactionStatusRequest) { r.PartyA = "" }},
		{"missing identifier", "IdentifierType", func(r *TransactionStatusRequest) { r.IdentifierType = "" }},
		{"missing remarks", "Remarks", func(r *TransactionStatusRequest) { r.Remarks = "" }},
		{"missing result url", "ResultURL", func(r *TransactionStatusRequest) { r.ResultURL = "" }},
		{"missing timeout url", "QueueTimeOutURL", func(r *TransactionStatusRequest) { r.QueueTimeOutURL = "" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base
			tc.mutate(&req)
			assertValidationField(t, req.Validate(), tc.field)
		})
	}
}

func TestC2BRegisterValidate(t *testing.T) {
	base := C2BRegisterRequest{
		ShortCode:       "174379",
		ResponseType:    ResponseTypeCompleted,
		ConfirmationURL: testCallbackURL,
		ValidationURL:   testCallbackURL,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	tests := []struct {
		name   string
		field  string
		mutate func(*C2BRegisterRequest)
	}{
		{"missing short code", "ShortCode", func(r *C2BRegisterRequest) { r.ShortCode = "" }},
		{"unknown response type", "ResponseType", func(r *C2BRegisterRequest) { r.ResponseType = "Maybe" }},
		{"missing confirmation url", "ConfirmationURL", func(r *C2BRegisterRequest) { r.ConfirmationURL = "" }},
		{"missing validation url", "ValidationURL", func(r *C2BRegisterRequest) { r.ValidationURL = "" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base
			tc.mutate(&req)
			assertValidationField(t, req.Validate(), tc.field)
		})
	}
}

func TestC2BSimulateValidate(t *testing.T) {
	base := C2BSimulateRequest{
		ShortCode:     "174379",
		CommandID:     CommandIDCustomerPayBillOnline,
		Amount:        10,
		MSISDN:        "254708374149",
		BillRefNumber: "ref",
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	tests := []struct {
		name   string
		field  string
		mutate func(*C2BSimulateRequest)
	}{
		{"missing short code", "ShortCode", func(r *C2BSimulateRequest) { r.ShortCode = "" }},
		{"unknown command", "CommandID", func(r *C2BSimulateRequest) { r.CommandID = "Nope" }},
		{"zero amount", "Amount", func(r *C2BSimulateRequest) { r.Amount = 0 }},
		{"bad msisdn", "MSISDN", func(r *C2BSimulateRequest) { r.MSISDN = "12345" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base
			tc.mutate(&req)
			assertValidationField(t, req.Validate(), tc.field)
		})
	}
}

func TestQRCodeValidate(t *testing.T) {
	base := QRCodeRequest{
		MerchantName: "Shop",
		RefNo:        "INV-1",
		Amount:       10,
		TrxCode:      QRTrxCodePayBill,
		CPI:          "174379",
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	tests := []struct {
		name   string
		field  string
		mutate func(*QRCodeRequest)
	}{
		{"missing merchant", "MerchantName", func(r *QRCodeRequest) { r.MerchantName = "" }},
		{"missing reference", "RefNo", func(r *QRCodeRequest) { r.RefNo = "" }},
		{"negative amount", "Amount", func(r *QRCodeRequest) { r.Amount = -1 }},
		{"unknown trx code", "TrxCode", func(r *QRCodeRequest) { r.TrxCode = "XX" }},
		{"missing cpi", "CPI", func(r *QRCodeRequest) { r.CPI = "" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base
			tc.mutate(&req)
			assertValidationField(t, req.Validate(), tc.field)
		})
	}
}

func TestSTKPushQueryValidate(t *testing.T) {
	base := STKPushQueryRequest{
		BusinessShortCode: "174379",
		Password:          "password",
		Timestamp:         "20240101120000",
		CheckoutRequestID: "ws_CO_191220191020363925",
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	tests := []struct {
		name   string
		field  string
		mutate func(*STKPushQueryRequest)
	}{
		{"missing short code", "BusinessShortCode", func(r *STKPushQueryRequest) { r.BusinessShortCode = "" }},
		{"missing password", "Password", func(r *STKPushQueryRequest) { r.Password = "" }},
		{"bad timestamp", "Timestamp", func(r *STKPushQueryRequest) { r.Timestamp = "2024" }},
		{"missing checkout id", "CheckoutRequestID", func(r *STKPushQueryRequest) { r.CheckoutRequestID = "" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base
			tc.mutate(&req)
			assertValidationField(t, req.Validate(), tc.field)
		})
	}
}

func TestSTKPushValidateFieldOrder(t *testing.T) {
	// Timestamp and transaction type are validated too.
	req := STKPushRequest{
		BusinessShortCode: "174379",
		Password:          "password",
		Timestamp:         "2024",
		TransactionType:   TransactionTypeCustomerPayBillOnline,
		Amount:            1,
		PhoneNumber:       "254712345678",
		CallBackURL:       testCallbackURL,
		AccountReference:  "ref",
		TransactionDesc:   "desc",
	}
	assertValidationField(t, req.Validate(), "Timestamp")

	req.Timestamp = "20240101120000"
	req.TransactionType = "Nope"
	assertValidationField(t, req.Validate(), "TransactionType")

	req.TransactionType = TransactionTypeCustomerPayBillOnline
	req.PartyA = "12345"
	assertValidationField(t, req.Validate(), "PartyA")
}

// ---------------------------------------------------------------------------
// Options and helpers
// ---------------------------------------------------------------------------

func TestOptionsApply(t *testing.T) {
	httpClient := &http.Client{}
	client, err := New("key", "secret", PRODUCTION,
		WithHTTPClient(httpClient),
		WithTimeout(5*time.Second),
		WithUserAgent("custom-agent/1.0"),
		WithRetries(3),
		WithRetryBackoff(10*time.Millisecond),
		WithShortCode("600000"),
		WithPassKey("pass-key"),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.BaseURL() != ProductionBaseURL {
		t.Errorf("BaseURL() = %q", client.BaseURL())
	}
	if client.http.Timeout != 5*time.Second {
		t.Errorf("timeout = %v", client.http.Timeout)
	}
	if client.http == httpClient {
		t.Error("WithTimeout should configure a copy of the HTTP client")
	}
	if client.userAgent != "custom-agent/1.0" {
		t.Errorf("userAgent = %q", client.userAgent)
	}
	if client.maxRetries != 3 {
		t.Errorf("maxRetries = %d", client.maxRetries)
	}
	if client.retryBackoff != 10*time.Millisecond {
		t.Errorf("retryBackoff = %v", client.retryBackoff)
	}
	if client.ShortCode() != "600000" {
		t.Errorf("ShortCode() = %q", client.ShortCode())
	}
	if client.passKey != "pass-key" {
		t.Errorf("passKey = %q", client.passKey)
	}
}

func TestBackoffGrowsAndIsCapped(t *testing.T) {
	client := &Client{retryBackoff: 10 * time.Millisecond}
	previous := time.Duration(0)
	for attempt := 1; attempt <= 10; attempt++ {
		delay := client.backoff(attempt)
		if delay < 0 || delay > maxRetryBackoff {
			t.Fatalf("backoff(%d) = %v, outside [0, %v]", attempt, delay, maxRetryBackoff)
		}
		if delay < previous/2 {
			t.Fatalf("backoff(%d) = %v shrank from %v", attempt, delay, previous)
		}
		previous = delay
	}

	noBackoff := &Client{}
	if got := noBackoff.backoff(1); got != 0 {
		t.Errorf("backoff with zero base = %v, want 0", got)
	}
}

func TestDecodeResponseError(t *testing.T) {
	client, _ := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	_, err := client.STKPush(context.Background(), validSTKPushRequest())
	if err == nil || !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("error = %v, want a decode error", err)
	}
}

func TestKeyValueInt64(t *testing.T) {
	if got := (KeyValue{Key: "k", Value: json.RawMessage("42")}).Int64(); got != 42 {
		t.Errorf("Int64() = %d, want 42", got)
	}
	if got := (KeyValue{Key: "k", Value: json.RawMessage(`"42"`)}).Float64(); got != 42 {
		t.Errorf("Float64() = %v, want 42", got)
	}
}

func TestParseResultCallbackWithoutEnvelope(t *testing.T) {
	payload := `{"ResultType":0,"ResultCode":0,"ResultDesc":"ok","TransactionID":"NLJ41HAY6Q"}`
	callback, err := ParseResultCallback([]byte(payload))
	if err != nil {
		t.Fatalf("ParseResultCallback: %v", err)
	}
	if callback.TransactionID != "NLJ41HAY6Q" {
		t.Errorf("TransactionID = %q", callback.TransactionID)
	}
}
