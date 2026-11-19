package mpesa

import (
	"context"
	"strings"
)

// pathB2C is the business to customer payment endpoint.
const pathB2C = "/mpesa/b2c/v1/paymentrequest"

// B2C command ids.
const (
	// CommandIDBusinessPayment pays a customer (e.g. a refund or payout).
	CommandIDBusinessPayment = "BusinessPayment"
	// CommandIDSalaryPayment pays a salary to a customer.
	CommandIDSalaryPayment = "SalaryPayment"
	// CommandIDPromotionPayment pays a promotional amount to a customer.
	CommandIDPromotionPayment = "PromotionPayment"
)

// B2CRequest is a business to customer payment request.
//
// SecurityCredential can be left empty when the client is configured with
// WithCertificate and WithInitiatorPassword, in which case it is generated for
// you.
type B2CRequest struct {
	// OriginatorConversationID is an optional id used to correlate the request
	// with the callback.
	OriginatorConversationID string `json:"OriginatorConversationID,omitempty"`
	// InitiatorName is the API operator name configured on the Daraja portal.
	InitiatorName string `json:"InitiatorName,omitempty"`
	// SecurityCredential is the encrypted initiator password.
	SecurityCredential string `json:"SecurityCredential,omitempty"`
	// CommandID is CommandIDBusinessPayment, CommandIDSalaryPayment or
	// CommandIDPromotionPayment.
	CommandID string `json:"CommandID,omitempty"`
	// Amount is the amount to pay, in whole Kenya Shillings.
	Amount int64 `json:"Amount"`
	// PartyA is the short code sending the money.
	PartyA string `json:"PartyA,omitempty"`
	// PartyB is the customer MSISDN receiving the money.
	PartyB string `json:"PartyB,omitempty"`
	// Remarks describes the transaction.
	Remarks string `json:"Remarks,omitempty"`
	// QueueTimeOutURL receives the notification when the request times out.
	QueueTimeOutURL string `json:"QueueTimeOutURL,omitempty"`
	// ResultURL receives the final result.
	ResultURL string `json:"ResultURL,omitempty"`
	// Occasion is an optional description. Daraja spells the JSON field
	// "Occassion", so that is what is sent on the wire.
	Occasion string `json:"Occassion,omitempty"`
}

// Validate reports the first invalid field of the request.
func (r B2CRequest) Validate() error {
	if err := requireField("InitiatorName", r.InitiatorName); err != nil {
		return err
	}
	if err := requireField("SecurityCredential", r.SecurityCredential); err != nil {
		return &ValidationError{Field: "SecurityCredential", Reason: "is required: set SecurityCredential or configure the client with WithCertificate and WithInitiatorPassword"}
	}
	if err := requireOneOf("CommandID", r.CommandID,
		CommandIDBusinessPayment, CommandIDSalaryPayment, CommandIDPromotionPayment); err != nil {
		return err
	}
	if err := requireAmount("Amount", r.Amount); err != nil {
		return err
	}
	if err := requireField("PartyA", r.PartyA); err != nil {
		return err
	}
	if err := requirePhone("PartyB", r.PartyB); err != nil {
		return err
	}
	if err := requireField("Remarks", r.Remarks); err != nil {
		return err
	}
	if err := requireURL("QueueTimeOutURL", r.QueueTimeOutURL); err != nil {
		return err
	}
	return requireURL("ResultURL", r.ResultURL)
}

// B2CResponse is the response to a B2C payment request. The final result is
// delivered asynchronously to ResultURL and can be decoded with
// ParseResultCallback.
type B2CResponse struct {
	Meta

	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// B2C sends a business to customer payment.
func (c *Client) B2C(ctx context.Context, req B2CRequest) (*B2CResponse, error) {
	if req.PartyA == "" {
		req.PartyA = c.shortCode
	}
	if req.CommandID == "" {
		req.CommandID = CommandIDBusinessPayment
	}
	if strings.TrimSpace(req.PartyB) != "" {
		partyB, err := NormalizePhoneNumber(req.PartyB)
		if err != nil {
			return nil, err
		}
		req.PartyB = partyB
	}
	credential, err := c.withSecurityCredential(req.SecurityCredential)
	if err != nil {
		return nil, err
	}
	req.SecurityCredential = credential
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var res B2CResponse
	if err := c.doJSON(ctx, pathB2C, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
