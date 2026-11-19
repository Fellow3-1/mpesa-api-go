// Package mpesa is a dependency free Go client for the Safaricom Daraja
// (M-Pesa) API.
//
// The client covers the payment APIs most integrations need:
//
//   - OAuth token generation (cached until shortly before it expires)
//   - Lipa na M-Pesa (STK push) requests and STK push queries
//   - C2B URL registration and sandbox C2B simulation
//   - B2C and B2B payment requests
//   - Transaction reversal, account balance and transaction status
//   - Dynamic QR code generation
//
// A Client is safe for concurrent use by multiple goroutines and should be
// created once and reused for the lifetime of the process.
//
//	c, err := mpesa.New(os.Getenv("MPESA_APP_KEY"), os.Getenv("MPESA_APP_SECRET"), mpesa.DEV,
//		mpesa.WithPassKey(os.Getenv("MPESA_PASS_KEY")),
//		mpesa.WithShortCode(os.Getenv("MPESA_SHORT_CODE")),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	res, err := c.STKPush(ctx, mpesa.STKPushRequest{
//		Amount:           100,
//		PhoneNumber:      "0712345678",
//		CallBackURL:      "https://example.com/mpesa/callback",
//		AccountReference: "INV-001",
//		TransactionDesc:  "Invoice 001",
//	})
//
// Callbacks sent by Safaricom can be decoded with ParseSTKCallback,
// ParseC2BConfirmation and ParseResultCallback.
package mpesa
