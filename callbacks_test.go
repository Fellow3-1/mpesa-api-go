package mpesa

import (
	"encoding/json"
	"testing"
	"time"
)

const stkSuccessCallback = `{
  "Body": {
    "stkCallback": {
      "MerchantRequestID": "29115-34620561-1",
      "CheckoutRequestID": "ws_CO_191220191020363925",
      "ResultCode": 0,
      "ResultDesc": "The service request is processed successfully.",
      "CallbackMetadata": {
        "Item": [
          {"Name": "Amount", "Value": 1},
          {"Name": "MpesaReceiptNumber", "Value": "NLJ7RT61SV"},
          {"Name": "TransactionDate", "Value": 20191219102115},
          {"Name": "PhoneNumber", "Value": 254708374149}
        ]
      }
    }
  }
}`

func TestParseSTKCallbackSuccess(t *testing.T) {
	callback, err := ParseSTKCallback([]byte(stkSuccessCallback))
	if err != nil {
		t.Fatalf("ParseSTKCallback: %v", err)
	}
	if !callback.Succeeded() {
		t.Errorf("Succeeded() = false, result code %d", callback.ResultCode)
	}
	if callback.Cancelled() || callback.TimedOut() {
		t.Error("callback should not be cancelled or timed out")
	}
	if callback.CheckoutRequestID != "ws_CO_191220191020363925" {
		t.Errorf("CheckoutRequestID = %q", callback.CheckoutRequestID)
	}
	if callback.Receipt() != "NLJ7RT61SV" {
		t.Errorf("Receipt() = %q", callback.Receipt())
	}
	if callback.Metadata == nil {
		t.Fatal("Metadata is nil")
	}
	if callback.Metadata.Amount != 1 {
		t.Errorf("Amount = %d", callback.Metadata.Amount)
	}
	if callback.Metadata.MpesaReceiptNumber != "NLJ7RT61SV" {
		t.Errorf("MpesaReceiptNumber = %q", callback.Metadata.MpesaReceiptNumber)
	}
	if callback.Metadata.PhoneNumber != "254708374149" {
		t.Errorf("PhoneNumber = %q", callback.Metadata.PhoneNumber)
	}
	want := time.Date(2019, time.December, 19, 10, 21, 15, 0, EAT())
	if !callback.Metadata.TransactionDate.Equal(want) {
		t.Errorf("TransactionDate = %v, want %v", callback.Metadata.TransactionDate, want)
	}
	if len(callback.Metadata.Items) != 4 {
		t.Errorf("Items = %d, want 4", len(callback.Metadata.Items))
	}
}

func TestParseSTKCallbackFailure(t *testing.T) {
	payload := `{"Body":{"stkCallback":{"MerchantRequestID":"29115-34620561-1","CheckoutRequestID":"ws_CO_191220191020363925","ResultCode":1032,"ResultDesc":"Request cancelled by user"}}}`
	callback, err := ParseSTKCallback([]byte(payload))
	if err != nil {
		t.Fatalf("ParseSTKCallback: %v", err)
	}
	if callback.Succeeded() {
		t.Error("Succeeded() = true for a cancelled payment")
	}
	if !callback.Cancelled() {
		t.Error("Cancelled() = false")
	}
	if callback.Metadata != nil {
		t.Error("Metadata should be nil for a failed payment")
	}
	if callback.Receipt() != "" {
		t.Errorf("Receipt() = %q, want empty", callback.Receipt())
	}
}

func TestParseSTKCallbackWithoutEnvelope(t *testing.T) {
	payload := `{"MerchantRequestID":"29115-34620561-1","CheckoutRequestID":"ws_CO_191220191020363925","ResultCode":1037,"ResultDesc":"DS timeout user cannot be reached"}`
	callback, err := ParseSTKCallback([]byte(payload))
	if err != nil {
		t.Fatalf("ParseSTKCallback: %v", err)
	}
	if !callback.TimedOut() {
		t.Error("TimedOut() = false")
	}
}

func TestParseSTKCallbackRejectsOtherPayloads(t *testing.T) {
	if _, err := ParseSTKCallback([]byte(`{"hello":"world"}`)); err == nil {
		t.Error("expected an error for a non-STK payload")
	}
	if _, err := ParseSTKCallback([]byte(`not json`)); err == nil {
		t.Error("expected an error for invalid JSON")
	}
}

func TestParseResultCallbackSuccess(t *testing.T) {
	payload := `{
	  "Result": {
	    "ResultType": 0,
	    "ResultCode": 0,
	    "ResultDesc": "The service request is processed successfully.",
	    "OriginatorConversationID": "10571-7910404-1",
	    "ConversationID": "AG_20191219_00004e4e5e5a0b0c1",
	    "TransactionID": "NLJ41HAY6Q",
	    "ResultParameters": {
	      "ResultParameter": [
	        {"Key": "TransactionAmount", "Value": 10},
	        {"Key": "TransactionReceipt", "Value": "NLJ41HAY6Q"},
	        {"Key": "ReceiverPartyPublicName", "Value": "254708374149 - John Doe"}
	      ]
	    },
	    "ReferenceData": {
	      "ReferenceItem": {"Key": "QueueTimeoutURL", "Value": "https://example.com/timeout"}
	    }
	  }
	}`

	callback, err := ParseResultCallback([]byte(payload))
	if err != nil {
		t.Fatalf("ParseResultCallback: %v", err)
	}
	if !callback.Succeeded() {
		t.Error("Succeeded() = false")
	}
	if callback.TimedOut() {
		t.Error("TimedOut() = true")
	}
	if callback.TransactionID != "NLJ41HAY6Q" {
		t.Errorf("TransactionID = %q", callback.TransactionID)
	}
	if got, ok := callback.Param("TransactionReceipt"); !ok || got != "NLJ41HAY6Q" {
		t.Errorf("Param(TransactionReceipt) = %q, %v", got, ok)
	}
	if amount, ok := callback.ParamFloat("TransactionAmount"); !ok || amount != 10 {
		t.Errorf("ParamFloat(TransactionAmount) = %v, %v", amount, ok)
	}
	if got, ok := callback.Param("QueueTimeoutURL"); !ok || got != "https://example.com/timeout" {
		t.Errorf("Param(QueueTimeoutURL) = %q, %v", got, ok)
	}
	if _, ok := callback.Param("Missing"); ok {
		t.Error("Param returned a value for a missing key")
	}
}

func TestParseResultCallbackTimeout(t *testing.T) {
	payload := `{"Result":{"ResultType":1,"ResultCode":0,"ResultDesc":"The service request timed out.","OriginatorConversationID":"10571-7910404-1","ConversationID":"AG_20191219_00004e4e5e5a0b0c1","TransactionID":""}}`
	callback, err := ParseResultCallback([]byte(payload))
	if err != nil {
		t.Fatalf("ParseResultCallback: %v", err)
	}
	if !callback.TimedOut() {
		t.Error("TimedOut() = false for ResultType 1")
	}
	if _, ok := callback.Param("TransactionReceipt"); ok {
		t.Error("timeout callback should not have result parameters")
	}
}

func TestParseResultCallbackReferenceItemArray(t *testing.T) {
	payload := `{"Result":{"ResultType":0,"ResultCode":0,"ResultDesc":"ok","TransactionID":"X","ReferenceData":{"ReferenceItem":[{"Key":"a","Value":"b"}]}}}`
	callback, err := ParseResultCallback([]byte(payload))
	if err != nil {
		t.Fatalf("ParseResultCallback: %v", err)
	}
	if got, ok := callback.Param("a"); !ok || got != "b" {
		t.Errorf("Param(a) = %q, %v", got, ok)
	}
}

func TestParseResultCallbackRejectsOtherPayloads(t *testing.T) {
	if _, err := ParseResultCallback([]byte(`{"Result":{}}`)); err == nil {
		t.Error("expected an error for an empty result")
	}
	if _, err := ParseResultCallback([]byte(`not json`)); err == nil {
		t.Error("expected an error for invalid JSON")
	}
}

func TestParseC2BConfirmation(t *testing.T) {
	payload := `{
	  "TransactionType": "Pay Bill",
	  "TransID": "RKTQDM7W6S",
	  "TransTime": "20191122063845",
	  "TransAmount": "10",
	  "BusinessShortCode": "600638",
	  "BillRefNumber": "account-1",
	  "InvoiceNumber": "",
	  "OrgAccountBalance": "49197.00",
	  "ThirdPartyTransID": "",
	  "MSISDN": "254708374149",
	  "FirstName": "John",
	  "MiddleName": "",
	  "LastName": "Doe"
	}`

	confirmation, err := ParseC2BConfirmation([]byte(payload))
	if err != nil {
		t.Fatalf("ParseC2BConfirmation: %v", err)
	}
	if confirmation.TransID != "RKTQDM7W6S" {
		t.Errorf("TransID = %q", confirmation.TransID)
	}
	if confirmation.BillRefNumber != "account-1" {
		t.Errorf("BillRefNumber = %q", confirmation.BillRefNumber)
	}
	if confirmation.MSISDN != "254708374149" {
		t.Errorf("MSISDN = %q", confirmation.MSISDN)
	}
	amount, err := confirmation.Amount()
	if err != nil {
		t.Fatalf("Amount: %v", err)
	}
	if amount != 10 {
		t.Errorf("Amount = %v, want 10", amount)
	}
	if got := confirmation.CustomerName(); got != "John Doe" {
		t.Errorf("CustomerName() = %q", got)
	}
	if len(confirmation.Raw) == 0 {
		t.Error("Raw is empty")
	}
}

func TestParseC2BConfirmationWithNumericFields(t *testing.T) {
	payload := `{"TransID":"X1","TransAmount":150.5,"MSISDN":254708374149,"BusinessShortCode":600638}`
	confirmation, err := ParseC2BConfirmation([]byte(payload))
	if err != nil {
		t.Fatalf("ParseC2BConfirmation: %v", err)
	}
	if confirmation.MSISDN != "254708374149" {
		t.Errorf("MSISDN = %q", confirmation.MSISDN)
	}
	amount, err := confirmation.Amount()
	if err != nil {
		t.Fatalf("Amount: %v", err)
	}
	if amount != 150.5 {
		t.Errorf("Amount = %v, want 150.5", amount)
	}
}

func TestParseC2BConfirmationRejectsOtherPayloads(t *testing.T) {
	if _, err := ParseC2BConfirmation([]byte(`{"foo":"bar"}`)); err == nil {
		t.Error("expected an error for a non-C2B payload")
	}
}

func TestCallbackAcks(t *testing.T) {
	ack := SuccessAck()
	if ack.ResultCode != 0 || ack.ResultDesc != "Success" {
		t.Errorf("SuccessAck() = %+v", ack)
	}

	reject := FailureAck("unknown account")
	if reject.ResultCode != 1 || reject.ResultDesc != "unknown account" {
		t.Errorf("FailureAck() = %+v", reject)
	}
	if FailureAck("").ResultDesc != "Rejected" {
		t.Error("FailureAck with an empty reason should default to Rejected")
	}

	encoded, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(encoded) != `{"ResultCode":0,"ResultDesc":"Success"}` {
		t.Errorf("encoded ack = %s", encoded)
	}
}

func TestCallbackItemAccessors(t *testing.T) {
	item := CallbackItem{Name: "TransactionDate", Value: json.RawMessage("20191219102115")}
	want := time.Date(2019, time.December, 19, 10, 21, 15, 0, EAT())
	if got := item.Time(); !got.Equal(want) {
		t.Errorf("Time() = %v, want %v", got, want)
	}
	if got := item.Int64(); got != 20191219102115 {
		t.Errorf("Int64() = %d", got)
	}

	quoted := CallbackItem{Name: "MpesaReceiptNumber", Value: json.RawMessage(`"NLJ7RT61SV"`)}
	if got := quoted.String(); got != "NLJ7RT61SV" {
		t.Errorf("String() = %q", got)
	}
}
