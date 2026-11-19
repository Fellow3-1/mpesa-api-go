package mpesa

import "context"

// pathAccountBalance is the account balance enquiry endpoint.
const pathAccountBalance = "/mpesa/accountbalance/v1/query"

// CommandIDAccountBalance requests the balance of a short code.
const CommandIDAccountBalance = "AccountBalance"

// AccountBalanceRequest asks Safaricom for the balance of a short code. The
// balance is delivered asynchronously to ResultURL.
type AccountBalanceRequest struct {
	// InitiatorName is the API operator name configured on the Daraja portal.
	InitiatorName string `json:"Initiator,omitempty"`
	// SecurityCredential is the encrypted initiator password.
	SecurityCredential string `json:"SecurityCredential,omitempty"`
	// CommandID is CommandIDAccountBalance and defaults to it.
	CommandID string `json:"CommandID,omitempty"`
	// PartyA is the short code whose balance is requested.
	PartyA string `json:"PartyA,omitempty"`
	// IdentifierType is the type of PartyA: "1" MSISDN, "2" till number, "4"
	// short code.
	IdentifierType string `json:"IdentifierType,omitempty"`
	// Remarks describes the enquiry.
	Remarks string `json:"Remarks,omitempty"`
	// QueueTimeOutURL receives the notification when the request times out.
	QueueTimeOutURL string `json:"QueueTimeOutURL,omitempty"`
	// ResultURL receives the balance.
	ResultURL string `json:"ResultURL,omitempty"`
}

// Validate reports the first invalid field of the request.
func (r AccountBalanceRequest) Validate() error {
	if err := requireField("InitiatorName", r.InitiatorName); err != nil {
		return err
	}
	if err := requireField("SecurityCredential", r.SecurityCredential); err != nil {
		return &ValidationError{Field: "SecurityCredential", Reason: "is required: set SecurityCredential or configure the client with WithCertificate and WithInitiatorPassword"}
	}
	if err := requireOneOf("CommandID", r.CommandID, CommandIDAccountBalance); err != nil {
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
	if err := requireURL("QueueTimeOutURL", r.QueueTimeOutURL); err != nil {
		return err
	}
	return requireURL("ResultURL", r.ResultURL)
}

// AccountBalanceResponse is the response to a balance enquiry. The balance
// itself arrives asynchronously on ResultURL.
type AccountBalanceResponse struct {
	Meta

	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// AccountBalance requests the balance of a short code.
func (c *Client) AccountBalance(ctx context.Context, req AccountBalanceRequest) (*AccountBalanceResponse, error) {
	if req.PartyA == "" {
		req.PartyA = c.shortCode
	}
	if req.CommandID == "" {
		req.CommandID = CommandIDAccountBalance
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

	var res AccountBalanceResponse
	if err := c.doJSON(ctx, pathAccountBalance, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
