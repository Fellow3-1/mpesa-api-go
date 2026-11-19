// Command demo exercises the mpesa package against the Safaricom sandbox.
//
// Configure it with environment variables (never hardcode credentials):
//
//	export MPESA_APP_KEY=...            # consumer key from the Daraja portal
//	export MPESA_APP_SECRET=...         # consumer secret
//	export MPESA_SHORT_CODE=174379      # sandbox pay bill
//	export MPESA_PASS_KEY=...           # Lipa na M-Pesa pass key
//	export MPESA_CALLBACK_URL=https://example.com/mpesa/callback
//	export MPESA_ENV=dev               # or production
//
// Optional, needed by B2C/B2B/reversal/balance/status:
//
//	export MPESA_INITIATOR_NAME=testapi
//	export MPESA_INITIATOR_PASSWORD=...
//	export MPESA_CERT_PATH=/path/to/SandboxCertificate.cer
//
// Then run a command, for example:
//
//	go run ./demo stk 10 0712345678
//
// Run with no arguments to see the list of commands.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	mpesa "github.com/Fellow3-1/mpesa-api-go"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("demo: ")

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	client, err := newClient()
	if err != nil {
		log.Fatalf("configure: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	command := os.Args[1]
	args := os.Args[2:]

	callbackURL := envOr("MPESA_CALLBACK_URL", "https://example.com/mpesa/callback")

	switch command {
	case "token":
		token, err := client.AccessToken(ctx)
		must(err)
		fmt.Println(token)

	case "stk":
		amount := argInt(args, 0, 10)
		phone := argString(args, 1, envOr("MPESA_PHONE", "254708374149"))
		res, err := client.STKPush(ctx, mpesa.STKPushRequest{
			Amount:           amount,
			PhoneNumber:      phone,
			CallBackURL:      callbackURL,
			AccountReference: "DEMO-" + strconv.FormatInt(amount, 10),
			TransactionDesc:  "Demo payment",
		})
		must(err)
		show("STK push accepted", res.Meta)

	case "query":
		id := argString(args, 0, "")
		res, err := client.STKPushQuery(ctx, mpesa.STKPushQueryRequest{CheckoutRequestID: id})
		must(err)
		show("STK push query", res.Meta)

	case "register-c2b":
		confirmation := argString(args, 0, callbackURL+"/confirm")
		validation := argString(args, 1, callbackURL+"/validate")
		res, err := client.RegisterC2BURLs(ctx, mpesa.C2BRegisterRequest{
			ConfirmationURL: confirmation,
			ValidationURL:   validation,
		})
		must(err)
		show("C2B URLs registered", res.Meta)

	case "simulate-c2b":
		amount := argInt(args, 0, 10)
		msisdn := argString(args, 1, "254708374149")
		reference := argString(args, 2, "DEMO")
		res, err := client.SimulateC2B(ctx, mpesa.C2BSimulateRequest{
			Amount:        amount,
			MSISDN:        msisdn,
			BillRefNumber: reference,
		})
		must(err)
		show("C2B simulation", res.Meta)

	case "b2c":
		amount := argInt(args, 0, 10)
		phone := argString(args, 1, "254708374149")
		remarks := argString(args, 2, "Demo payout")
		res, err := client.B2C(ctx, mpesa.B2CRequest{
			InitiatorName:   envOr("MPESA_INITIATOR_NAME", "testapi"),
			Amount:          amount,
			PartyB:          phone,
			Remarks:         remarks,
			ResultURL:       callbackURL + "/result",
			QueueTimeOutURL: callbackURL + "/timeout",
		})
		must(err)
		show("B2C request accepted", res.Meta)

	case "b2b":
		amount := argInt(args, 0, 10)
		partyB := argString(args, 1, "600000")
		reference := argString(args, 2, "DEMO")
		res, err := client.B2B(ctx, mpesa.B2BRequest{
			InitiatorName:          envOr("MPESA_INITIATOR_NAME", "testapi"),
			CommandID:              mpesa.CommandIDBusinessPayBill,
			Amount:                 amount,
			PartyB:                 partyB,
			AccountReference:       reference,
			Remarks:                "Demo B2B",
			ResultURL:              callbackURL + "/result",
			QueueTimeOutURL:        callbackURL + "/timeout",
			SenderIdentifierType:   "4",
			ReceiverIdentifierType: "4",
		})
		must(err)
		show("B2B request accepted", res.Meta)

	case "reversal":
		transactionID := argString(args, 0, "")
		amount := argInt(args, 1, 10)
		res, err := client.Reversal(ctx, mpesa.ReversalRequest{
			InitiatorName:   envOr("MPESA_INITIATOR_NAME", "testapi"),
			TransactionID:   transactionID,
			Amount:          amount,
			ReceiverParty:   envOr("MPESA_SHORT_CODE", "600000"),
			ResultURL:       callbackURL + "/result",
			QueueTimeOutURL: callbackURL + "/timeout",
			Remarks:         "Demo reversal",
		})
		must(err)
		show("Reversal accepted", res.Meta)

	case "balance":
		res, err := client.AccountBalance(ctx, mpesa.AccountBalanceRequest{
			InitiatorName:   envOr("MPESA_INITIATOR_NAME", "testapi"),
			Remarks:         "Demo balance",
			ResultURL:       callbackURL + "/result",
			QueueTimeOutURL: callbackURL + "/timeout",
		})
		must(err)
		show("Balance request accepted", res.Meta)

	case "status":
		transactionID := argString(args, 0, "")
		res, err := client.TransactionStatus(ctx, mpesa.TransactionStatusRequest{
			InitiatorName:   envOr("MPESA_INITIATOR_NAME", "testapi"),
			TransactionID:   transactionID,
			Remarks:         "Demo status",
			ResultURL:       callbackURL + "/result",
			QueueTimeOutURL: callbackURL + "/timeout",
		})
		must(err)
		show("Status request accepted", res.Meta)

	case "qr":
		amount := argInt(args, 0, 10)
		reference := argString(args, 1, "DEMO-1")
		res, err := client.GenerateQRCode(ctx, mpesa.QRCodeRequest{
			MerchantName: "Demo Merchant",
			RefNo:        reference,
			Amount:       amount,
		})
		must(err)
		show("QR code generated", res.Meta)

	default:
		usage()
		os.Exit(2)
	}
}

// newClient builds a client from the environment.
func newClient() (*mpesa.Client, error) {
	env := mpesa.DEV
	switch strings.ToLower(envOr("MPESA_ENV", "dev")) {
	case "production", "prod", "live":
		env = mpesa.PRODUCTION
	case "dev", "sandbox", "test":
		env = mpesa.DEV
	default:
		return nil, fmt.Errorf("MPESA_ENV must be dev or production")
	}

	options := []mpesa.Option{
		mpesa.WithShortCode(os.Getenv("MPESA_SHORT_CODE")),
		mpesa.WithPassKey(os.Getenv("MPESA_PASS_KEY")),
		mpesa.WithInitiatorPassword(os.Getenv("MPESA_INITIATOR_PASSWORD")),
		mpesa.WithRetries(2),
	}

	if path := os.Getenv("MPESA_CERT_PATH"); path != "" {
		pemBytes, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read MPESA_CERT_PATH: %w", err)
		}
		cert, err := mpesa.LoadCertificate(pemBytes)
		if err != nil {
			return nil, err
		}
		options = append(options, mpesa.WithCertificate(cert))
	}

	return mpesa.New(os.Getenv("MPESA_APP_KEY"), os.Getenv("MPESA_APP_SECRET"), env, options...)
}

// show prints an API response, preferring the raw JSON returned by Safaricom.
func show(label string, meta mpesa.Meta) {
	fmt.Printf("%s (http %d):\n", label, meta.StatusCode)
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, meta.Raw, "", "  "); err == nil {
		fmt.Println(pretty.String())
		return
	}
	fmt.Println(string(meta.Raw))
}

func must(err error) {
	if err == nil {
		return
	}
	if apiErr, ok := mpesa.AsAPIError(err); ok {
		log.Fatalf("%v (status %d)", err, apiErr.StatusCode)
	}
	log.Fatal(err)
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func argString(args []string, index int, fallback string) string {
	if index < len(args) && strings.TrimSpace(args[index]) != "" {
		return args[index]
	}
	return fallback
}

func argInt(args []string, index int, fallback int64) int64 {
	if index >= len(args) {
		return fallback
	}
	value, err := strconv.ParseInt(args[index], 10, 64)
	if err != nil {
		log.Fatalf("argument %d must be a whole number, got %q", index+1, args[index])
	}
	return value
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage: demo <command> [args]

Commands:
  token                                   Print an OAuth access token
  stk <amount> <phone>                    Send an STK push prompt
  query <checkoutRequestID>               Query an STK push
  register-c2b [confirmURL] [validateURL] Register C2B URLs
  simulate-c2b <amount> <msisdn> <ref>    Simulate a C2B payment (sandbox)
  b2c <amount> <phone> [remarks]          Send a B2C payment
  b2b <amount> <partyB> [accountRef]      Send a B2B payment
  reversal <transactionID> <amount>       Reverse a transaction
  balance                                 Request the account balance
  status <transactionID>                  Request a transaction status
  qr <amount> [refNo]                     Generate a dynamic QR code

Environment:
  MPESA_APP_KEY, MPESA_APP_SECRET, MPESA_ENV (dev|production),
  MPESA_SHORT_CODE, MPESA_PASS_KEY, MPESA_CALLBACK_URL, MPESA_PHONE,
  MPESA_INITIATOR_NAME, MPESA_INITIATOR_PASSWORD, MPESA_CERT_PATH
`)
}
