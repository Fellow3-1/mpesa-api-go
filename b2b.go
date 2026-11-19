package mpesa

import "context"

// pathB2B is the business to business payment endpoint.
const pathB2B = "/mpesa/b2b/v1/paymentrequest"

// B2B command ids.
const (
	// CommandIDBusinessPayBill pays another pay bill.
	CommandIDBusinessPayBill = "BusinessPayBill"
	// CommandIDBusinessBuyGoods buys goods from a till number.
	CommandIDBusinessBuyGoods = "BusinessBuyGoods"
	// CommandIDMerchantToMerchantTransfer moves money between merchants.
	CommandIDMerchantToMerchantTransfer = "MerchantToMerchantTransfer"
)

// B2BRequest is a business to business payment request.
type B2BRequest struct {
	// InitiatorName is the API operator name configured on the Daraja portal.
	InitiatorName string `json:"Initiator,omitempty"`
	// SecurityCredential is the encrypted initiator password.
	SecurityCredential string `json:"SecurityCredential,omitempty"`
	// CommandID is CommandIDBusinessPayBill, CommandIDBusinessBuyGoods or
	// CommandIDMerchantToMerchantTransfer.
	CommandID string `json:"CommandID,omitempty"`
	// SenderIdentifierType is the type of PartyA: "1" MSISDN, "2" till number,
	// "4" short code.
	SenderIdentifierType string `json:"SenderIdentifierType,omitempty"`
	// ReceiverIdentifierType is the type of PartyB. Daraja spells the JSON field
	// "RecieverIdentifierType", so that is what is sent on the wire.
	ReceiverIdentifierType string `json:"RecieverIdentifierType,omitempty"`
	// Amount is the amount to pay, in whole Kenya Shillings.
	Amount int64 `json:"Amount"`
	// PartyA is the sending short code, till number or MSISDN.
	PartyA string `json:"PartyA,omitempty"`
	// PartyB is the receiving short code, till number or MSISDN.
	PartyB string `json:"PartyB,omitempty"`
	// AccountReference is the account number at the receiving organisation.
	AccountReference string `json:"AccountReference,omitempty"`
	// Remarks describes the transaction.
	Remarks string `json:"Remarks,omitempty"`
	// QueueTimeOutURL receives the notification when the request times out.
	QueueTimeOutURL string `json:"QueueTimeOutURL,omitempty"`
	// ResultURL receives the final result.
	ResultURL string `json:"ResultURL,omitempty"`
}

// Validate reports the first invalid field of the request.
func (r B2BRequest) Validate() error {
	if err := requireField("InitiatorName", r.InitiatorName); err != nil {
		return err
	}
	if err := requireField("SecurityCredential", r.SecurityCredential); err != nil {
		return &ValidationError{Field: "SecurityCredential", Reason: "is required: set SecurityCredential or configure the client with WithCertificate and WithInitiatorPassword"}
	}
	if err := requireOneOf("CommandID", r.CommandID,
		CommandIDBusinessPayBill, CommandIDBusinessBuyGoods, CommandIDMerchantToMerchantTransfer); err != nil {
		return err
	}
	if err := requireField("SenderIdentifierType", r.SenderIdentifierType); err != nil {
		return err
	}
	if err := requireField("ReceiverIdentifierType", r.ReceiverIdentifierType); err != nil {
		return err
	}
	if err := requireAmount("Amount", r.Amount); err != nil {
		return err
	}
	if err := requireField("PartyA", r.PartyA); err != nil {
		return err
	}
	if err := requireField("PartyB", r.PartyB); err != nil {
		return err
	}
	if err := requireField("AccountReference", r.AccountReference); err != nil {
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

// B2BResponse is the response to a B2B payment request.
type B2BResponse struct {
	Meta

	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// B2B sends a business to business payment.
func (c *Client) B2B(ctx context.Context, req B2BRequest) (*B2BResponse, error) {
	if req.PartyA == "" {
		req.PartyA = c.shortCode
	}
	if req.CommandID == "" {
		req.CommandID = CommandIDBusinessPayBill
	}
	if req.SenderIdentifierType == "" {
		req.SenderIdentifierType = "4"
	}
	if req.ReceiverIdentifierType == "" {
		req.ReceiverIdentifierType = "4"
	}
	credential, err := c.withSecurityCredential(req.SecurityCredential)
	if err != nil {
		return nil, err
	}
	req.SecurityCredential = credential
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var res B2BResponse
	if err := c.doJSON(ctx, pathB2B, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
