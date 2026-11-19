package mpesa

import "context"

// pathReversal is the transaction reversal endpoint.
const pathReversal = "/mpesa/reversal/v1/request"

// CommandIDTransactionReversal reverses a completed transaction.
const CommandIDTransactionReversal = "TransactionReversal"

// ReversalRequest reverses a completed M-Pesa transaction.
type ReversalRequest struct {
	// InitiatorName is the API operator name configured on the Daraja portal.
	InitiatorName string `json:"Initiator,omitempty"`
	// SecurityCredential is the encrypted initiator password.
	SecurityCredential string `json:"SecurityCredential,omitempty"`
	// CommandID is CommandIDTransactionReversal and defaults to it.
	CommandID string `json:"CommandID,omitempty"`
	// TransactionID is the M-Pesa receipt number to reverse.
	TransactionID string `json:"TransactionID,omitempty"`
	// Amount is the amount to reverse, in whole Kenya Shillings.
	Amount int64 `json:"Amount"`
	// ReceiverParty is the short code, till number or MSISDN that received the
	// money.
	ReceiverParty string `json:"ReceiverParty,omitempty"`
	// ReceiverIdentifierType is the type of ReceiverParty. Daraja spells the
	// JSON field "RecieverIdentifierType".
	ReceiverIdentifierType string `json:"RecieverIdentifierType,omitempty"`
	// ResultURL receives the final result.
	ResultURL string `json:"ResultURL,omitempty"`
	// QueueTimeOutURL receives the notification when the request times out.
	QueueTimeOutURL string `json:"QueueTimeOutURL,omitempty"`
	// Remarks describes the reversal.
	Remarks string `json:"Remarks,omitempty"`
	// Occasion is an optional description. Daraja spells the JSON field
	// "Occassion".
	Occasion string `json:"Occassion,omitempty"`
}

// Validate reports the first invalid field of the request.
func (r ReversalRequest) Validate() error {
	if err := requireField("InitiatorName", r.InitiatorName); err != nil {
		return err
	}
	if err := requireField("SecurityCredential", r.SecurityCredential); err != nil {
		return &ValidationError{Field: "SecurityCredential", Reason: "is required: set SecurityCredential or configure the client with WithCertificate and WithInitiatorPassword"}
	}
	if err := requireOneOf("CommandID", r.CommandID, CommandIDTransactionReversal); err != nil {
		return err
	}
	if err := requireField("TransactionID", r.TransactionID); err != nil {
		return err
	}
	if err := requireAmount("Amount", r.Amount); err != nil {
		return err
	}
	if err := requireField("ReceiverParty", r.ReceiverParty); err != nil {
		return err
	}
	if err := requireField("ReceiverIdentifierType", r.ReceiverIdentifierType); err != nil {
		return err
	}
	if err := requireURL("ResultURL", r.ResultURL); err != nil {
		return err
	}
	if err := requireURL("QueueTimeOutURL", r.QueueTimeOutURL); err != nil {
		return err
	}
	return requireField("Remarks", r.Remarks)
}

// ReversalResponse is the response to a reversal request.
type ReversalResponse struct {
	Meta

	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// Reversal reverses a completed M-Pesa transaction.
func (c *Client) Reversal(ctx context.Context, req ReversalRequest) (*ReversalResponse, error) {
	if req.CommandID == "" {
		req.CommandID = CommandIDTransactionReversal
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

	var res ReversalResponse
	if err := c.doJSON(ctx, pathReversal, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
