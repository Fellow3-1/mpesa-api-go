package mpesa

import "context"

// pathQRCode is the dynamic QR code endpoint.
const pathQRCode = "/mpesa/qrcode/v1/generate"

// QR transaction codes.
const (
	// QRTrxCodeBuyGoods is a buy goods (till) QR code.
	QRTrxCodeBuyGoods = "BG"
	// QRTrxCodePayBill is a pay bill QR code.
	QRTrxCodePayBill = "PB"
	// QRTrxCodeWithdraw is a withdrawal QR code.
	QRTrxCodeWithdraw = "WA"
	// QRTrxCodeSendMoney is a send money QR code.
	QRTrxCodeSendMoney = "SM"
	// QRTrxCodeSendToBusiness is a send to business QR code.
	QRTrxCodeSendToBusiness = "SB"
)

// QRCodeRequest generates a dynamic M-Pesa QR code.
type QRCodeRequest struct {
	// MerchantName is the name displayed in the customer's M-Pesa app.
	MerchantName string `json:"MerchantName,omitempty"`
	// RefNo is the reference for the transaction.
	RefNo string `json:"RefNo,omitempty"`
	// Amount is the amount to charge, in whole Kenya Shillings. Zero means the
	// customer enters the amount.
	Amount int64 `json:"Amount,omitempty"`
	// TrxCode is QRTrxCodeBuyGoods, QRTrxCodePayBill, QRTrxCodeWithdraw,
	// QRTrxCodeSendMoney or QRTrxCodeSendToBusiness.
	TrxCode string `json:"TrxCode,omitempty"`
	// CPI is the credit party identifier (the short code).
	CPI string `json:"CPI,omitempty"`
	// Size is the QR image size, for example "200".
	Size string `json:"Size,omitempty"`
}

// Validate reports the first invalid field of the request.
func (r QRCodeRequest) Validate() error {
	if err := requireField("MerchantName", r.MerchantName); err != nil {
		return err
	}
	if err := requireField("RefNo", r.RefNo); err != nil {
		return err
	}
	if r.Amount < 0 {
		return &ValidationError{Field: "Amount", Reason: "must not be negative"}
	}
	if err := requireOneOf("TrxCode", r.TrxCode,
		QRTrxCodeBuyGoods, QRTrxCodePayBill, QRTrxCodeWithdraw, QRTrxCodeSendMoney, QRTrxCodeSendToBusiness); err != nil {
		return err
	}
	return requireField("CPI", r.CPI)
}

// QRCodeResponse is the response to a QR code request.
type QRCodeResponse struct {
	Meta

	RequestID           string `json:"RequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	QRCode              string `json:"QRCode"`
}

// GenerateQRCode creates a dynamic M-Pesa QR code that customers can scan.
func (c *Client) GenerateQRCode(ctx context.Context, req QRCodeRequest) (*QRCodeResponse, error) {
	if req.CPI == "" {
		req.CPI = c.shortCode
	}
	if req.TrxCode == "" {
		req.TrxCode = QRTrxCodeBuyGoods
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var res QRCodeResponse
	if err := c.doJSON(ctx, pathQRCode, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
