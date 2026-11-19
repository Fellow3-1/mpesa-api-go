package mpesa_test

import (
	"context"
	"fmt"
	"log"
	"os"

	mpesa "github.com/Fellow3-1/mpesa-api-go"
)

// ExampleNew shows the typical client setup. Keep credentials in the
// environment or a secret manager, never in source control.
func ExampleNew() {
	client, err := mpesa.New(
		os.Getenv("MPESA_APP_KEY"),
		os.Getenv("MPESA_APP_SECRET"),
		mpesa.DEV, // mpesa.PRODUCTION for live payments
		mpesa.WithShortCode(os.Getenv("MPESA_SHORT_CODE")),
		mpesa.WithPassKey(os.Getenv("MPESA_PASS_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client
}

// ExampleClient_STKPush sends a Lipa na M-Pesa prompt to a customer.
func ExampleClient_STKPush() {
	client, err := mpesa.New("app-key", "app-secret", mpesa.DEV,
		mpesa.WithShortCode("174379"),
		mpesa.WithPassKey("pass-key"),
	)
	if err != nil {
		log.Fatal(err)
	}

	res, err := client.STKPush(context.Background(), mpesa.STKPushRequest{
		Amount:           100,
		PhoneNumber:      "0712345678", // normalised to 254712345678
		CallBackURL:      "https://example.com/mpesa/callback",
		AccountReference: "INV-001",
		TransactionDesc:  "Invoice 001",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.CheckoutRequestID, res.ResponseDescription)
}

// ExampleParseSTKCallback decodes the callback Safaricom posts when the
// customer responds to an STK push.
func ExampleParseSTKCallback() {
	payload := []byte(`{
	  "Body": {
	    "stkCallback": {
	      "MerchantRequestID": "29115-34620561-1",
	      "CheckoutRequestID": "ws_CO_191220191020363925",
	      "ResultCode": 0,
	      "ResultDesc": "The service request is processed successfully.",
	      "CallbackMetadata": {
	        "Item": [
	          {"Name": "Amount", "Value": 100},
	          {"Name": "MpesaReceiptNumber", "Value": "NLJ7RT61SV"},
	          {"Name": "TransactionDate", "Value": 20191219102115},
	          {"Name": "PhoneNumber", "Value": 254708374149}
	        ]
	      }
	    }
	  }
	}`)

	callback, err := mpesa.ParseSTKCallback(payload)
	if err != nil {
		log.Fatal(err)
	}
	if !callback.Succeeded() {
		fmt.Println("payment failed:", callback.ResultDesc)
		return
	}
	fmt.Printf("paid %d by %s, receipt %s\n",
		callback.Metadata.Amount, callback.Metadata.PhoneNumber, callback.Receipt())

	// A handler should answer Safaricom with the success acknowledgement.
	fmt.Printf("%+v\n", mpesa.SuccessAck())

	// Output:
	// paid 100 by 254708374149, receipt NLJ7RT61SV
	// {ResultCode:0 ResultDesc:Success}
}

// ExampleNormalizePhoneNumber shows the phone number formats the client
// accepts.
func ExampleNormalizePhoneNumber() {
	for _, input := range []string{"0712345678", "+254 712 345 678", "712345678"} {
		normalized, err := mpesa.NormalizePhoneNumber(input)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(normalized)
	}

	// Output:
	// 254712345678
	// 254712345678
	// 254712345678
}

// ExampleClient_B2C pays a customer from a business short code.
func ExampleClient_B2C() {
	client, err := mpesa.New("app-key", "app-secret", mpesa.DEV,
		mpesa.WithShortCode("600000"),
		mpesa.WithInitiatorPassword(os.Getenv("MPESA_INITIATOR_PASSWORD")),
	)
	if err != nil {
		log.Fatal(err)
	}

	res, err := client.B2C(context.Background(), mpesa.B2CRequest{
		InitiatorName:   "testapi",
		Amount:          250,
		PartyB:          "0712345678",
		Remarks:         "Refund",
		ResultURL:       "https://example.com/mpesa/result",
		QueueTimeOutURL: "https://example.com/mpesa/timeout",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.ConversationID)
}
