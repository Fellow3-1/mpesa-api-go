package mpesa

import "context"

// pathTransactionStatus is the transaction status endpoint.
const pathTransactionStatus = "/mpesa/transactionstatus/v1/query"

// CommandIDTransactionStatus queries the status of a transaction.
const CommandIDTransactionStatus = "TransactionStatus"

// TransactionStatusRequest asks Safaricom for the status of a transaction. The
// result is delivered asynchronously to ResultURL.
type TransactionStatusRequest struct {
	// InitiatorName is the API operator name configured on the Daraja portal.
	InitiatorName string `json:"Initiator,omitempty"`
	// SecurityCredential is the encrypted initiator password.
	SecurityCredential string `json:"SecurityCredential,omitempty"`
	// CommandID is CommandIDTransactionStatus and defaults to it.
	CommandID string `json:"CommandID,omitempty"`
	// TransactionID is the M-Pesa receipt number being queried.
	TransactionID string `json:"TransactionID,omitempty"`
	// PartyA is the short code or MSISDN that owns the transaction.
	PartyA string `json:"PartyA,omitempty"`
	// IdentifierType is the type of PartyA: "1" MSISDN, "2" till number, "4"
	// short code.
	IdentifierType string `json:"IdentifierType,omitempty"`
	// ResultURL receives the result.
	ResultURL string `json:"ResultURL,omitempty"`
	// QueueTimeOutURL receives the notification when the request times out.
	QueueTimeOutURL string `json:"QueueTimeOutURL,omitempty"`
	// Remarks describes the enquiry.
	Remarks string `json:"Remarks,omitempty"`
	// Occasion is an optional description. Daraja spells the JSON field
	// "Occassion".
	Occasion string `json:"Occassion,omitempty"`
}

// Validate reports the first invalid field of the request.
func (r TransactionStatusRequest) Validate() error {
	if err := requireField("InitiatorName", r.InitiatorName); err != nil {
		return err
	}
	if err := requireField("SecurityCredential", r.SecurityCredential); err != nil {
		return &ValidationError{Field: "SecurityCredential", Reason: "is required: set SecurityCredential or configure the client with WithCertificate and WithInitiatorPassword"}
	}
	if err := requireOneOf("CommandID", r.CommandID, CommandIDTransactionStatus); err != nil {
		return err
	}
	if err := requireField("TransactionID", r.TransactionID); err != nil {
		return err
	}
	if err := requireField("PartyA", r.PartyA); err != nil {
		return err
	}
	if err := requireField("IdentifierType", r.IdentifierType); err != nil {
		return err
	}
	if err := requireField("Remarks", r.Remarks); err != nil {
		return err
	}
	if err := requireURL("ResultURL", r.ResultURL); err != nil {
		return err
	}
	return requireURL("QueueTimeOutURL", r.QueueTimeOutURL)
}

// TransactionStatusResponse is the response to a transaction status enquiry.
type TransactionStatusResponse struct {
	Meta

	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// TransactionStatus enquires about the status of a transaction.
func (c *Client) TransactionStatus(ctx context.Context, req TransactionStatusRequest) (*TransactionStatusResponse, error) {
	if req.PartyA == "" {
		req.PartyA = c.shortCode
	}
	if req.CommandID == "" {
		req.CommandID = CommandIDTransactionStatus
	}
	if req.IdentifierType == "" {
		req.IdentifierType = "4"
	}
	credential, err := c.withSecurityCredential(req.SecurityCredential)
	if err != nil {
		return nil, err
	}
	req.SecurityCredential = credential
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var res TransactionStatusResponse
	if err := c.doJSON(ctx, pathTransactionStatus, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
