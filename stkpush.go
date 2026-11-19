package mpesa

import (
	"context"
	"strings"
)

const (
	// pathSTKPush is the Lipa na M-Pesa (STK push) endpoint.
	pathSTKPush = "/mpesa/stkpush/v1/processrequest"
	// pathSTKQuery is the STK push query endpoint.
	pathSTKQuery = "/mpesa/stkpushquery/v1/query"
)

// STK push transaction types.
const (
	// TransactionTypeCustomerPayBillOnline is used when the short code is a
	// pay bill number.
	TransactionTypeCustomerPayBillOnline = "CustomerPayBillOnline"
	// TransactionTypeCustomerBuyGoodsOnline is used when the short code is a
	// till number (Buy Goods).
	TransactionTypeCustomerBuyGoodsOnline = "CustomerBuyGoodsOnline"
)

// STKPushRequest is the payload for a Lipa na M-Pesa online (STK push)
// request.
//
// BusinessShortCode, Password and Timestamp default to the values configured on
// the Client (WithShortCode and WithPassKey). PartyA and PartyB default to
// PhoneNumber and BusinessShortCode.
type STKPushRequest struct {
	// BusinessShortCode is the pay bill or till number receiving the payment.
	BusinessShortCode string `json:"BusinessShortCode,omitempty"`
	// Password is base64(shortCode + passKey + timestamp). It is generated
	// automatically when the client has a pass key.
	Password string `json:"Password,omitempty"`
	// Timestamp is the request time in YYYYMMDDHHmmss (EAT). It defaults to
	// now.
	Timestamp string `json:"Timestamp,omitempty"`
	// TransactionType is TransactionTypeCustomerPayBillOnline (default) or
	// TransactionTypeCustomerBuyGoodsOnline.
	TransactionType string `json:"TransactionType,omitempty"`
	// Amount is the amount to charge, in whole Kenya Shillings.
	Amount int64 `json:"Amount"`
	// PartyA is the paying MSISDN. It defaults to PhoneNumber.
	PartyA string `json:"PartyA,omitempty"`
	// PartyB is the receiving short code. It defaults to BusinessShortCode.
	PartyB string `json:"PartyB,omitempty"`
	// PhoneNumber is the MSISDN that receives the prompt. It is normalised to
	// 2547XXXXXXXX before the request is sent.
	PhoneNumber string `json:"PhoneNumber,omitempty"`
	// CallBackURL receives the asynchronous payment result.
	CallBackURL string `json:"CallBackURL,omitempty"`
	// AccountReference is the account or invoice number shown to the customer.
	// Daraja documents a 12 character limit for pay bill payments.
	AccountReference string `json:"AccountReference,omitempty"`
	// TransactionDesc describes the transaction on the customer's statement.
	// Daraja documents a 13 character limit.
	TransactionDesc string `json:"TransactionDesc,omitempty"`
}

// Validate reports the first invalid field of the request.
func (r STKPushRequest) Validate() error {
	if err := requireField("BusinessShortCode", r.BusinessShortCode); err != nil {
		return err
	}
	if err := requireField("Password", r.Password); err != nil {
		return &ValidationError{Field: "Password", Reason: "is required: set Password or configure the client with WithPassKey and WithShortCode"}
	}
	if err := requireTimestamp("Timestamp", r.Timestamp); err != nil {
		return err
	}
	if err := requireOneOf("TransactionType", r.TransactionType,
		TransactionTypeCustomerPayBillOnline, TransactionTypeCustomerBuyGoodsOnline); err != nil {
		return err
	}
	if err := requireAmount("Amount", r.Amount); err != nil {
		return err
	}
	if err := requirePhone("PhoneNumber", r.PhoneNumber); err != nil {
		return err
	}
	if r.PartyA != "" {
		if err := requirePhone("PartyA", r.PartyA); err != nil {
			return err
		}
	}
	if err := requireURL("CallBackURL", r.CallBackURL); err != nil {
		return err
	}
	if err := requireField("AccountReference", r.AccountReference); err != nil {
		return err
	}
	return requireField("TransactionDesc", r.TransactionDesc)
}

// STKPushResponse is the response to an STK push request.
type STKPushResponse struct {
	Meta

	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

// Accepted reports whether Safaricom accepted the request and sent a prompt to
// the customer. The payment result arrives asynchronously on the callback URL.
func (r *STKPushResponse) Accepted() bool { return r.ResponseCode == "0" }

// STKPush initiates a Lipa na M-Pesa online (STK push) payment.
//
// The customer receives a prompt and the result is posted to
// STKPushRequest.CallBackURL; decode it with ParseSTKCallback.
func (c *Client) STKPush(ctx context.Context, req STKPushRequest) (*STKPushResponse, error) {
	// Fill in the values Daraja can derive from the client configuration.
	if req.BusinessShortCode == "" {
		req.BusinessShortCode = c.shortCode
	}
	if req.Timestamp == "" {
		req.Timestamp = FormatTimestamp(c.now())
	}
	if req.Password == "" && c.passKey != "" && req.BusinessShortCode != "" {
		req.Password = GenerateSTKPushPassword(req.BusinessShortCode, c.passKey, req.Timestamp)
	}
	if req.TransactionType == "" {
		req.TransactionType = TransactionTypeCustomerPayBillOnline
	}
	if strings.TrimSpace(req.PhoneNumber) == "" {
		req.PhoneNumber = req.PartyA
	}
	if strings.TrimSpace(req.PhoneNumber) != "" {
		phone, err := NormalizePhoneNumber(req.PhoneNumber)
		if err != nil {
			return nil, err
		}
		req.PhoneNumber = phone
	}
	if strings.TrimSpace(req.PartyA) != "" {
		partyA, err := NormalizePhoneNumber(req.PartyA)
		if err != nil {
			return nil, err
		}
		req.PartyA = partyA
	}
	if req.PartyA == "" {
		req.PartyA = req.PhoneNumber
	}
	if req.PartyB == "" {
		req.PartyB = req.BusinessShortCode
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var res STKPushResponse
	if err := c.doJSON(ctx, pathSTKPush, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// STKPushQueryRequest is the payload for querying the status of an STK push.
type STKPushQueryRequest struct {
	// BusinessShortCode is the short code that initiated the push.
	BusinessShortCode string `json:"BusinessShortCode,omitempty"`
	// Password is base64(shortCode + passKey + timestamp).
	Password string `json:"Password,omitempty"`
	// Timestamp is the query time in YYYYMMDDHHmmss (EAT).
	Timestamp string `json:"Timestamp,omitempty"`
	// CheckoutRequestID is the id from the STK push response.
	CheckoutRequestID string `json:"CheckoutRequestID,omitempty"`
}

// Validate reports the first invalid field of the request.
func (r STKPushQueryRequest) Validate() error {
	if err := requireField("BusinessShortCode", r.BusinessShortCode); err != nil {
		return err
	}
	if err := requireField("Password", r.Password); err != nil {
		return &ValidationError{Field: "Password", Reason: "is required: set Password or configure the client with WithPassKey and WithShortCode"}
	}
	if err := requireTimestamp("Timestamp", r.Timestamp); err != nil {
		return err
	}
	return requireField("CheckoutRequestID", r.CheckoutRequestID)
}

// STKPushQueryResponse is the response to an STK push query.
type STKPushQueryResponse struct {
	Meta

	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	ResultCode          string `json:"ResultCode"`
	ResultDesc          string `json:"ResultDesc"`
}

// Succeeded reports whether the queried transaction completed successfully.
func (r *STKPushQueryResponse) Succeeded() bool { return r.ResultCode == "0" }

// Cancelled reports whether the customer cancelled the prompt.
func (r *STKPushQueryResponse) Cancelled() bool { return r.ResultCode == "1032" }

// TimedOut reports whether the prompt expired or the phone could not be
// reached.
func (r *STKPushQueryResponse) TimedOut() bool { return r.ResultCode == "1037" }

// Pending reports whether Safaricom has not reached a final state yet. Daraja
// answers a query that arrives too early with no ResultCode and an error such
// as "The transaction is still under processing".
func (r *STKPushQueryResponse) Pending() bool {
	return r.ResultCode == "" && !r.Succeeded()
}

// STKPushQuery returns the status of an STK push that has not called back yet.
//
// The query is only useful after a short delay: an immediate query usually
// reports the transaction as pending.
func (c *Client) STKPushQuery(ctx context.Context, req STKPushQueryRequest) (*STKPushQueryResponse, error) {
	if req.BusinessShortCode == "" {
		req.BusinessShortCode = c.shortCode
	}
	if req.Timestamp == "" {
		req.Timestamp = FormatTimestamp(c.now())
	}
	if req.Password == "" && c.passKey != "" && req.BusinessShortCode != "" {
		req.Password = GenerateSTKPushPassword(req.BusinessShortCode, c.passKey, req.Timestamp)
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var res STKPushQueryResponse
	if err := c.doJSON(ctx, pathSTKQuery, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
