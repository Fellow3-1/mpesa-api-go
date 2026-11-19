package mpesa

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// STK push
// ---------------------------------------------------------------------------

func TestSTKPushBuildsRequest(t *testing.T) {
	fixed := time.Date(2024, time.July, 1, 9, 30, 0, 0, time.UTC)
	client, rec := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, stkSuccessPayload())
	}))
	client.now = func() time.Time { return fixed }

	req := STKPushRequest{
		Amount:           100,
		PhoneNumber:      "0712 345 678",
		CallBackURL:      testCallbackURL,
		AccountReference: "INV-001",
		TransactionDesc:  "Invoice 001",
	}
	res, err := client.STKPush(context.Background(), req)
	if err != nil {
		t.Fatalf("STKPush: %v", err)
	}

	request := rec.last()
	if request.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", request.Method)
	}
	if request.Path != "/mpesa/stkpush/v1/processrequest" {
		t.Errorf("path = %s", request.Path)
	}
	if got := request.Header.Get("Authorization"); got != "Bearer token" {
		t.Errorf("Authorization = %q", got)
	}
	if got := request.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := request.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q", got)
	}
	if ua := request.Header.Get("User-Agent"); ua != "mpesa-api-go/"+Version {
		t.Errorf("User-Agent = %q", ua)
	}

	body := decodeBody(t, request.Body)
	timestamp := FormatTimestamp(fixed)
	assertString(t, body, "BusinessShortCode", testShortCode)
	assertString(t, body, "Timestamp", timestamp)
	assertString(t, body, "Password", GenerateSTKPushPassword(testShortCode, testPassKey, timestamp))
	assertString(t, body, "TransactionType", TransactionTypeCustomerPayBillOnline)
	assertNumber(t, body, "Amount", 100)
	assertString(t, body, "PhoneNumber", "254712345678")
	assertString(t, body, "PartyA", "254712345678")
	assertString(t, body, "PartyB", testShortCode)
	assertString(t, body, "CallBackURL", testCallbackURL)
	assertString(t, body, "AccountReference", "INV-001")
	assertString(t, body, "TransactionDesc", "Invoice 001")

	if res.StatusCode != http.StatusOK {
		t.Errorf("Meta.StatusCode = %d", res.StatusCode)
	}
	if res.CheckoutRequestID != "ws_CO_191220191020363925" {
		t.Errorf("CheckoutRequestID = %q", res.CheckoutRequestID)
	}
	if res.MerchantRequestID != "29115-34620561-1" {
		t.Errorf("MerchantRequestID = %q", res.MerchantRequestID)
	}
	if !res.Accepted() {
		t.Error("Accepted() = false")
	}
	if len(res.Raw) == 0 {
		t.Error("Meta.Raw is empty")
	}
}

func TestSTKPushHonoursExplicitValues(t *testing.T) {
	client, rec := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, stkSuccessPayload())
	}))

	req := STKPushRequest{
		BusinessShortCode: "600000",
		Password:          "custom-password",
		Timestamp:         "20240101120000",
		TransactionType:   TransactionTypeCustomerBuyGoodsOnline,
		Amount:            250,
		PartyA:            "254700000001",
		PartyB:            "600000",
		PhoneNumber:       "254700000001",
		CallBackURL:       testCallbackURL,
		AccountReference:  "ORDER-9",
		TransactionDesc:   "Order 9",
	}
	if _, err := client.STKPush(context.Background(), req); err != nil {
		t.Fatalf("STKPush: %v", err)
	}

	body := decodeBody(t, rec.last().Body)
	assertString(t, body, "BusinessShortCode", "600000")
	assertString(t, body, "Password", "custom-password")
	assertString(t, body, "Timestamp", "20240101120000")
	assertString(t, body, "TransactionType", TransactionTypeCustomerBuyGoodsOnline)
	assertString(t, body, "PartyB", "600000")
}

func TestSTKPushNormalizesPartyA(t *testing.T) {
	client, rec := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, stkSuccessPayload())
	}))

	req := validSTKPushRequest()
	req.PhoneNumber = ""
	req.PartyA = "0712 345 678"
	if _, err := client.STKPush(context.Background(), req); err != nil {
		t.Fatalf("STKPush: %v", err)
	}

	body := decodeBody(t, rec.last().Body)
	assertString(t, body, "PartyA", "254712345678")
	assertString(t, body, "PhoneNumber", "254712345678")
}

func TestSTKPushValidation(t *testing.T) {
	client, _ := New("key", "secret", DEV, WithBaseURL("http://127.0.0.1:1"), WithShortCode(testShortCode))
	_, err := client.STKPush(context.Background(), validSTKPushRequest())
	assertValidationField(t, err, "Password")
}

func TestSTKPushValidationFields(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request, _ []byte) {
		t.Error("no request should be made for an invalid payload")
	})

	tests := []struct {
		name   string
		mutate func(*STKPushRequest)
		field  string
	}{
		{"missing amount", func(r *STKPushRequest) { r.Amount = 0 }, "Amount"},
		{"missing callback", func(r *STKPushRequest) { r.CallBackURL = "" }, "CallBackURL"},
		{"bad callback", func(r *STKPushRequest) { r.CallBackURL = "not-a-url" }, "CallBackURL"},
		{"missing reference", func(r *STKPushRequest) { r.AccountReference = "" }, "AccountReference"},
		{"missing description", func(r *STKPushRequest) { r.TransactionDesc = "" }, "TransactionDesc"},
		{"bad phone", func(r *STKPushRequest) { r.PhoneNumber = "0812345678" }, "phone number"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validSTKPushRequest()
			tc.mutate(&req)
			_, err := client.STKPush(context.Background(), req)
			assertValidationField(t, err, tc.field)
		})
	}
}

func TestSTKPushQueryDefaults(t *testing.T) {
	client, rec := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, map[string]any{
			"MerchantRequestID":   "29115-34620561-1",
			"CheckoutRequestID":   "ws_CO_191220191020363925",
			"ResponseCode":        "0",
			"ResponseDescription": "The service request has been accepted successsfully",
			"ResultCode":          "1032",
			"ResultDesc":          "Request cancelled by user",
		})
	}))

	res, err := client.STKPushQuery(context.Background(), STKPushQueryRequest{
		CheckoutRequestID: "ws_CO_191220191020363925",
	})
	if err != nil {
		t.Fatalf("STKPushQuery: %v", err)
	}

	body := decodeBody(t, rec.last().Body)
	assertString(t, body, "BusinessShortCode", testShortCode)
	assertString(t, body, "CheckoutRequestID", "ws_CO_191220191020363925")
	if pw, _ := body["Password"].(string); pw == "" {
		t.Error("Password was not generated")
	}
	if ts, _ := body["Timestamp"].(string); len(ts) != 14 {
		t.Errorf("Timestamp = %v", body["Timestamp"])
	}

	if res.ResultCode != "1032" {
		t.Errorf("ResultCode = %q", res.ResultCode)
	}
	if res.Succeeded() {
		t.Error("Succeeded() = true for a cancelled request")
	}
	if !res.Cancelled() {
		t.Error("Cancelled() = false for result code 1032")
	}
	if res.Pending() {
		t.Error("Pending() = true for a final result code")
	}
}

func TestSTKPushQueryPending(t *testing.T) {
	client, _ := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, map[string]any{
			"requestId":    "req-1",
			"errorCode":    "500.001.1001",
			"errorMessage": "The transaction is still under processing",
			"ResponseCode": "0",
		})
	}))

	res, err := client.STKPushQuery(context.Background(), STKPushQueryRequest{
		CheckoutRequestID: "ws_CO_191220191020363925",
	})
	if err != nil {
		t.Fatalf("STKPushQuery: %v", err)
	}
	if !res.Pending() {
		t.Error("Pending() = false for a query without a result code")
	}
	if res.Succeeded() {
		t.Error("Succeeded() = true for a pending query")
	}
}

// ---------------------------------------------------------------------------
// C2B
// ---------------------------------------------------------------------------

func TestRegisterC2BURLs(t *testing.T) {
	client, rec := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, map[string]any{"ResponseDescription": "Success"})
	}))

	res, err := client.RegisterC2BURLs(context.Background(), C2BRegisterRequest{
		ConfirmationURL: "https://example.com/mpesa/confirm",
		ValidationURL:   "https://example.com/mpesa/validate",
	})
	if err != nil {
		t.Fatalf("RegisterC2BURLs: %v", err)
	}
	if res.ResponseDescription != "Success" {
		t.Errorf("ResponseDescription = %q", res.ResponseDescription)
	}

	request := rec.last()
	if request.Path != "/mpesa/c2b/v1/registerurl" {
		t.Errorf("path = %s", request.Path)
	}
	body := decodeBody(t, request.Body)
	assertString(t, body, "ShortCode", testShortCode)
	assertString(t, body, "ResponseType", ResponseTypeCompleted)
	assertString(t, body, "ConfirmationURL", "https://example.com/mpesa/confirm")
	assertString(t, body, "ValidationURL", "https://example.com/mpesa/validate")
}

func TestSimulateC2B(t *testing.T) {
	client, rec := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ConversationID":           "AG_20191219_00004f0e4d3b1cb1b0b0",
			"OriginatorConversationID": "12345-6789-1",
			"ResponseDescription":      "Accept the service request successfully.",
		})
	}))

	res, err := client.SimulateC2B(context.Background(), C2BSimulateRequest{
		Amount:        10,
		MSISDN:        "0708 374 149",
		BillRefNumber: "account-1",
	})
	if err != nil {
		t.Fatalf("SimulateC2B: %v", err)
	}
	if res.ConversationID == "" {
		t.Error("ConversationID is empty")
	}

	request := rec.last()
	if request.Path != "/mpesa/c2b/v1/simulate" {
		t.Errorf("path = %s", request.Path)
	}
	body := decodeBody(t, request.Body)
	assertString(t, body, "ShortCode", testShortCode)
	assertString(t, body, "CommandID", CommandIDCustomerPayBillOnline)
	assertString(t, body, "Msisdn", "254708374149")
	assertString(t, body, "BillRefNumber", "account-1")
	assertNumber(t, body, "Amount", 10)
}

func TestSimulateC2BIsSandboxOnly(t *testing.T) {
	client, rec := newTestClientWithOptions(t, func(w http.ResponseWriter, r *http.Request, _ []byte) {
		t.Error("no request should be made in production")
	})
	client.env = PRODUCTION

	_, err := client.SimulateC2B(context.Background(), C2BSimulateRequest{Amount: 10, MSISDN: "254708374149"})
	if !errors.Is(err, ErrSandboxOnly) {
		t.Fatalf("error = %v, want ErrSandboxOnly", err)
	}
	if len(rec.all()) != 0 {
		t.Errorf("expected no HTTP requests, got %d", len(rec.all()))
	}
}

// ---------------------------------------------------------------------------
// B2C / B2B / reversal / balance / status
// ---------------------------------------------------------------------------

func TestB2CGeneratesSecurityCredential(t *testing.T) {
	cert, key, _ := testCertificate(t)
	client, rec := newTestClientWithOptions(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ConversationID":           "AG_20240101_00004f0e4d3b1cb1b0b0",
			"OriginatorConversationID": "12345-6789-1",
			"ResponseCode":             "0",
			"ResponseDescription":      "Accept the service request successfully.",
		})
	}), WithCertificate(cert), WithInitiatorPassword("Safaricom123!!"))

	res, err := client.B2C(context.Background(), B2CRequest{
		InitiatorName:   "testapi",
		Amount:          10,
		PartyB:          "0712345678",
		Remarks:         "payout",
		ResultURL:       testCallbackURL,
		QueueTimeOutURL: testCallbackURL,
		Occasion:        "payout-1",
	})
	if err != nil {
		t.Fatalf("B2C: %v", err)
	}
	if res.ConversationID == "" {
		t.Error("ConversationID is empty")
	}

	request := rec.last()
	if request.Path != "/mpesa/b2c/v1/paymentrequest" {
		t.Errorf("path = %s", request.Path)
	}
	body := decodeBody(t, request.Body)
	assertString(t, body, "InitiatorName", "testapi")
	assertString(t, body, "CommandID", CommandIDBusinessPayment)
	assertString(t, body, "PartyA", testShortCode)
	assertString(t, body, "PartyB", "254712345678")
	assertString(t, body, "Remarks", "payout")
	assertString(t, body, "Occassion", "payout-1")
	assertNumber(t, body, "Amount", 10)
	assertNoKey(t, body, "Initiator")

	credential, _ := body["SecurityCredential"].(string)
	if credential == "" {
		t.Fatal("SecurityCredential was not generated")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(credential)
	if err != nil {
		t.Fatalf("SecurityCredential is not base64: %v", err)
	}
	plaintext, err := rsa.DecryptPKCS1v15(nil, key, ciphertext)
	if err != nil {
		t.Fatalf("decrypt SecurityCredential: %v", err)
	}
	if string(plaintext) != "Safaricom123!!" {
		t.Errorf("decrypted credential = %q", plaintext)
	}
}

func TestB2CRequiresSecurityCredential(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request, _ []byte) {
		t.Error("no request should be made for an invalid payload")
	})
	_, err := client.B2C(context.Background(), B2CRequest{
		InitiatorName:   "testapi",
		Amount:          10,
		PartyB:          "254712345678",
		Remarks:         "payout",
		ResultURL:       testCallbackURL,
		QueueTimeOutURL: testCallbackURL,
	})
	assertValidationField(t, err, "SecurityCredential")
}

func TestB2BFieldNames(t *testing.T) {
	cert, _, _ := testCertificate(t)
	client, rec := newTestClientWithOptions(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ConversationID":           "AG_20240101_00004f0e4d3b1cb1b0b0",
			"OriginatorConversationID": "12345-6789-1",
			"ResponseCode":             "0",
			"ResponseDescription":      "Accept the service request successfully.",
		})
	}), WithCertificate(cert), WithInitiatorPassword("Safaricom123!!"))

	_, err := client.B2B(context.Background(), B2BRequest{
		InitiatorName:          "testapi",
		Amount:                 22,
		PartyB:                 "600000",
		AccountReference:       "his",
		Remarks:                "stock",
		ResultURL:              testCallbackURL,
		QueueTimeOutURL:        testCallbackURL,
		ReceiverIdentifierType: "2",
	})
	if err != nil {
		t.Fatalf("B2B: %v", err)
	}

	request := rec.last()
	if request.Path != "/mpesa/b2b/v1/paymentrequest" {
		t.Errorf("path = %s", request.Path)
	}
	body := decodeBody(t, request.Body)
	assertString(t, body, "Initiator", "testapi")
	assertString(t, body, "SenderIdentifierType", "4")
	assertString(t, body, "RecieverIdentifierType", "2")
	assertString(t, body, "CommandID", CommandIDBusinessPayBill)
	assertString(t, body, "AccountReference", "his")
	assertNoKey(t, body, "InitiatorName")
	assertNoKey(t, body, "ReceiverIdentifierType")
}

func TestReversal(t *testing.T) {
	cert, _, _ := testCertificate(t)
	client, rec := newTestClientWithOptions(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ConversationID":           "AG_20240101_00004f0e4d3b1cb1b0b0",
			"OriginatorConversationID": "12345-6789-1",
			"ResponseCode":             "0",
			"ResponseDescription":      "Accept the service request successfully.",
		})
	}), WithCertificate(cert), WithInitiatorPassword("Safaricom123!!"))

	_, err := client.Reversal(context.Background(), ReversalRequest{
		InitiatorName:   "testapi",
		TransactionID:   "NLJ7RT61SV",
		Amount:          2,
		ReceiverParty:   "600000",
		ResultURL:       testCallbackURL,
		QueueTimeOutURL: testCallbackURL,
		Remarks:         "wrong number",
		Occasion:        "reversal-1",
	})
	if err != nil {
		t.Fatalf("Reversal: %v", err)
	}

	request := rec.last()
	if request.Path != "/mpesa/reversal/v1/request" {
		t.Errorf("path = %s", request.Path)
	}
	body := decodeBody(t, request.Body)
	assertString(t, body, "CommandID", CommandIDTransactionReversal)
	assertString(t, body, "TransactionID", "NLJ7RT61SV")
	assertString(t, body, "RecieverIdentifierType", "4")
	assertString(t, body, "Occassion", "reversal-1")
	assertNumber(t, body, "Amount", 2)
}

func TestAccountBalance(t *testing.T) {
	cert, _, _ := testCertificate(t)
	client, rec := newTestClientWithOptions(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ConversationID":           "AG_20240101_00004f0e4d3b1cb1b0b0",
			"OriginatorConversationID": "12345-6789-1",
			"ResponseCode":             "0",
			"ResponseDescription":      "Accept the service request successfully.",
		})
	}), WithCertificate(cert), WithInitiatorPassword("Safaricom123!!"))

	_, err := client.AccountBalance(context.Background(), AccountBalanceRequest{
		InitiatorName:   "testapi",
		Remarks:         "balance",
		ResultURL:       testCallbackURL,
		QueueTimeOutURL: testCallbackURL,
	})
	if err != nil {
		t.Fatalf("AccountBalance: %v", err)
	}

	request := rec.last()
	if request.Path != "/mpesa/accountbalance/v1/query" {
		t.Errorf("path = %s", request.Path)
	}
	body := decodeBody(t, request.Body)
	assertString(t, body, "CommandID", CommandIDAccountBalance)
	assertString(t, body, "PartyA", testShortCode)
	assertString(t, body, "IdentifierType", "4")
	assertString(t, body, "Remarks", "balance")
}

func TestTransactionStatus(t *testing.T) {
	cert, _, _ := testCertificate(t)
	client, rec := newTestClientWithOptions(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ConversationID":           "AG_20240101_00004f0e4d3b1cb1b0b0",
			"OriginatorConversationID": "12345-6789-1",
			"ResponseCode":             "0",
			"ResponseDescription":      "Accept the service request successfully.",
		})
	}), WithCertificate(cert), WithInitiatorPassword("Safaricom123!!"))

	_, err := client.TransactionStatus(context.Background(), TransactionStatusRequest{
		InitiatorName:   "testapi",
		TransactionID:   "NLJ7RT61SV",
		Remarks:         "status",
		ResultURL:       testCallbackURL,
		QueueTimeOutURL: testCallbackURL,
	})
	if err != nil {
		t.Fatalf("TransactionStatus: %v", err)
	}

	request := rec.last()
	if request.Path != "/mpesa/transactionstatus/v1/query" {
		t.Errorf("path = %s", request.Path)
	}
	body := decodeBody(t, request.Body)
	assertString(t, body, "CommandID", CommandIDTransactionStatus)
	assertString(t, body, "TransactionID", "NLJ7RT61SV")
	assertString(t, body, "PartyA", testShortCode)
}

// ---------------------------------------------------------------------------
// QR codes
// ---------------------------------------------------------------------------

func TestGenerateQRCode(t *testing.T) {
	client, rec := newTestClient(t, withOAuth("token", func(w http.ResponseWriter, r *http.Request, _ []byte) {
		writeJSON(w, http.StatusOK, map[string]any{
			"RequestID":           "16740-34861180-1",
			"ResponseCode":        "00",
			"ResponseDescription": "QR Code Successfully Generated",
			"QRCode":              "iVBORw0KGgoAAAANSUhEUg",
		})
	}))

	res, err := client.GenerateQRCode(context.Background(), QRCodeRequest{
		MerchantName: "Test Merchant",
		RefNo:        "INV-001",
		Amount:       120,
	})
	if err != nil {
		t.Fatalf("GenerateQRCode: %v", err)
	}
	if res.QRCode == "" {
		t.Error("QRCode is empty")
	}

	request := rec.last()
	if request.Path != "/mpesa/qrcode/v1/generate" {
		t.Errorf("path = %s", request.Path)
	}
	body := decodeBody(t, request.Body)
	assertString(t, body, "MerchantName", "Test Merchant")
	assertString(t, body, "RefNo", "INV-001")
	assertString(t, body, "TrxCode", QRTrxCodeBuyGoods)
	assertString(t, body, "CPI", testShortCode)
	assertNumber(t, body, "Amount", 120)
}

func TestGenerateQRCodeValidation(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request, _ []byte) {
		t.Error("no request should be made for an invalid payload")
	})
	_, err := client.GenerateQRCode(context.Background(), QRCodeRequest{MerchantName: "Shop"})
	assertValidationField(t, err, "RefNo")
}

// assertValidationField asserts that err is a *ValidationError for field.
func assertValidationField(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected a validation error for %q, got nil", field)
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error %v is not a *ValidationError", err)
	}
	if validationErr.Field != field {
		t.Fatalf("validation field = %q, want %q (%v)", validationErr.Field, field, err)
	}
	if !strings.Contains(err.Error(), field) {
		t.Errorf("error message %q does not mention %q", err.Error(), field)
	}
}
