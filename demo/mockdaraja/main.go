// Command mockdaraja is a tiny stand-in for the Safaricom Daraja API. It lets
// you try the demo CLI and the library end to end without sandbox credentials.
//
//	go run ./demo/mockdaraja -addr 127.0.0.1:8099
//	MPESA_BASE_URL=http://127.0.0.1:8099 MPESA_APP_KEY=key MPESA_APP_SECRET=secret \
//	MPESA_SHORT_CODE=174379 MPESA_PASS_KEY=passkey go run ./demo stk 100 0712345678
//
// Responses are static and every request is logged, which makes it useful for
// eyeballing the exact payloads the client sends.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8099", "address to listen on")
	flag.Parse()

	mux := http.NewServeMux()

	mux.HandleFunc("/oauth/v1/generate", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{
			"access_token": "mock-access-token-8f3c1d",
			"expires_in":   "3599",
		})
	})

	mux.HandleFunc("/mpesa/stkpush/v1/processrequest", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"MerchantRequestID":   "29115-34620561-1",
			"CheckoutRequestID":   "ws_CO_191220191020363925",
			"ResponseCode":        "0",
			"ResponseDescription": "Success. Request accepted for processing",
			"CustomerMessage":     "Success. Request accepted for processing",
		})
	})

	mux.HandleFunc("/mpesa/stkpushquery/v1/query", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"MerchantRequestID":   "29115-34620561-1",
			"CheckoutRequestID":   "ws_CO_191220191020363925",
			"ResponseCode":        "0",
			"ResponseDescription": "The service request has been accepted successsfully",
			"ResultCode":          "0",
			"ResultDesc":          "The service request is processed successfully.",
		})
	})

	mux.HandleFunc("/mpesa/c2b/v1/registerurl", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"ResponseDescription": "Success"})
	})

	mux.HandleFunc("/mpesa/c2b/v1/simulate", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"ConversationID":           "AG_20191219_00004f0e4d3b1cb1b0b0",
			"OriginatorConversationID": "10571-7910404-1",
			"ResponseDescription":      "Accept the service request successfully.",
		})
	})

	mux.HandleFunc("/mpesa/b2c/v1/paymentrequest", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"ConversationID":           "AG_20191219_00004e4e5e5a0b0c1",
			"OriginatorConversationID": "10571-7910404-1",
			"ResponseCode":             "0",
			"ResponseDescription":      "Accept the service request successfully.",
		})
	})

	mux.HandleFunc("/mpesa/b2b/v1/paymentrequest", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"ConversationID":           "AG_20191219_00004f0e4d3b1cb1b0b0",
			"OriginatorConversationID": "10571-7910404-2",
			"ResponseCode":             "0",
			"ResponseDescription":      "Accept the service request successfully.",
		})
	})

	mux.HandleFunc("/mpesa/reversal/v1/request", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"ConversationID":           "AG_20191219_00004e4e5e5a0b0c1",
			"OriginatorConversationID": "10571-7910404-3",
			"ResponseCode":             "0",
			"ResponseDescription":      "Accept the service request successfully.",
		})
	})

	mux.HandleFunc("/mpesa/accountbalance/v1/query", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"ConversationID":           "AG_20191219_00004f0e4d3b1cb1b0b0",
			"OriginatorConversationID": "10571-7910404-4",
			"ResponseCode":             "0",
			"ResponseDescription":      "Accept the service request successfully.",
		})
	})

	mux.HandleFunc("/mpesa/transactionstatus/v1/query", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"ConversationID":           "AG_20191219_00004e4e5e5a0b0c1",
			"OriginatorConversationID": "10571-7910404-5",
			"ResponseCode":             "0",
			"ResponseDescription":      "Accept the service request successfully.",
		})
	})

	mux.HandleFunc("/mpesa/qrcode/v1/generate", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"RequestID":           "16740-34861180-1",
			"ResponseCode":        "00",
			"ResponseDescription": "QR Code Successfully Generated",
			"QRCode":              "iVBORw0KGgoAAAANSUhEUgAAASwAAAEsCAYAAAB5fY51AAAAAXNSR0IArs4c6QAAIABJREFUeF7t",
		})
	})

	log.Printf("mock Daraja API listening on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, logRequests(mux)))
}

// writeJSON writes a 200 response with a JSON body.
func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// logRequests logs method, path and whether the request carried a bearer token.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := "none"
		if header := r.Header.Get("Authorization"); header != "" {
			if len(header) > 13 {
				auth = header[:13] + "..."
			} else {
				auth = header
			}
		}
		log.Printf("%s %s auth=%s", r.Method, r.URL.RequestURI(), auth)
		next.ServeHTTP(w, r)
	})
}
