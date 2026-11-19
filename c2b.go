package mpesa

import (
	"context"
	"strings"
)

const (
	// pathC2BRegister is the C2B URL registration endpoint.
	pathC2BRegister = "/mpesa/c2b/v1/registerurl"
	// pathC2BSimulate is the sandbox only C2B simulation endpoint.
	pathC2BSimulate = "/mpesa/c2b/v1/simulate"
)

// C2B response types.
const (
	// ResponseTypeCompleted tells Safaricom to accept the payment.
	ResponseTypeCompleted = "Completed"
	// ResponseTypeCancelled tells Safaricom to reject the payment.
	ResponseTypeCancelled = "Cancelled"
)

// C2B command ids.
const (
	// CommandIDCustomerPayBillOnline is used for pay bill payments.
	CommandIDCustomerPayBillOnline = "CustomerPayBillOnline"
	// CommandIDCustomerBuyGoodsOnline is used for buy goods payments.
	CommandIDCustomerBuyGoodsOnline = "CustomerBuyGoodsOnline"
)

// C2BRegisterRequest registers the validation and confirmation URLs that
// Safaricom calls for customer to business payments.
type C2BRegisterRequest struct {
	// ShortCode is the pay bill or till number.
	ShortCode string `json:"ShortCode,omitempty"`
	// ResponseType is ResponseTypeCompleted (default) or
	// ResponseTypeCancelled and decides what happens when the validation URL
	// accepts the payment.
	ResponseType string `json:"ResponseType,omitempty"`
	// ConfirmationURL receives the final payment notification.
	ConfirmationURL string `json:"ConfirmationURL,omitempty"`
	// ValidationURL is called before the payment is accepted. Return an accept
	// or reject payload (see C2BValidationResult).
	ValidationURL string `json:"ValidationURL,omitempty"`
}

// Validate reports the first invalid field of the request.
func (r C2BRegisterRequest) Validate() error {
	if err := requireField("ShortCode", r.ShortCode); err != nil {
		return err
	}
	if err := requireOneOf("ResponseType", r.ResponseType, ResponseTypeCompleted, ResponseTypeCancelled); err != nil {
		return err
	}
	if err := requireURL("ConfirmationURL", r.ConfirmationURL); err != nil {
		return err
	}
	return requireURL("ValidationURL", r.ValidationURL)
}

// C2BRegisterResponse is the response to a C2B URL registration.
type C2BRegisterResponse struct {
	Meta

	ResponseDescription string `json:"ResponseDescription"`
}

// RegisterC2BURLs registers the C2B validation and confirmation URLs for a
// short code.
func (c *Client) RegisterC2BURLs(ctx context.Context, req C2BRegisterRequest) (*C2BRegisterResponse, error) {
	if req.ShortCode == "" {
		req.ShortCode = c.shortCode
	}
	if req.ResponseType == "" {
		req.ResponseType = ResponseTypeCompleted
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var res C2BRegisterResponse
	if err := c.doJSON(ctx, pathC2BRegister, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// C2BSimulateRequest simulates a customer to business payment. It is only
// available in the sandbox.
type C2BSimulateRequest struct {
	// ShortCode is the pay bill or till number that receives the payment.
	ShortCode string `json:"ShortCode,omitempty"`
	// CommandID is CommandIDCustomerPayBillOnline (default) or
	// CommandIDCustomerBuyGoodsOnline.
	CommandID string `json:"CommandID,omitempty"`
	// Amount is the simulated amount in whole Kenya Shillings.
	Amount int64 `json:"Amount"`
	// MSISDN is the simulated customer phone number.
	MSISDN string `json:"Msisdn,omitempty"`
	// BillRefNumber is the account number the customer pays to.
	BillRefNumber string `json:"BillRefNumber,omitempty"`
}

// Validate reports the first invalid field of the request.
func (r C2BSimulateRequest) Validate() error {
	if err := requireField("ShortCode", r.ShortCode); err != nil {
		return err
	}
	if err := requireOneOf("CommandID", r.CommandID,
		CommandIDCustomerPayBillOnline, CommandIDCustomerBuyGoodsOnline); err != nil {
		return err
	}
	if err := requireAmount("Amount", r.Amount); err != nil {
		return err
	}
	return requirePhone("MSISDN", r.MSISDN)
}

// C2BSimulateResponse is the response to a simulated C2B payment.
type C2BSimulateResponse struct {
	Meta

	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// SimulateC2B simulates a C2B payment. It returns ErrSandboxOnly in
// production.
func (c *Client) SimulateC2B(ctx context.Context, req C2BSimulateRequest) (*C2BSimulateResponse, error) {
	if c.env == PRODUCTION {
		return nil, ErrSandboxOnly
	}
	if req.ShortCode == "" {
		req.ShortCode = c.shortCode
	}
	if req.CommandID == "" {
		req.CommandID = CommandIDCustomerPayBillOnline
	}
	if strings.TrimSpace(req.MSISDN) != "" {
		msisdn, err := NormalizePhoneNumber(req.MSISDN)
		if err != nil {
			return nil, err
		}
		req.MSISDN = msisdn
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var res C2BSimulateResponse
	if err := c.doJSON(ctx, pathC2BSimulate, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
