# Screenshots

These images are captures of real runs of this repository, not mockups.

| Image | What it shows | Transcript |
| --- | --- | --- |
| [`demo-stk-push.png`](demo-stk-push.png) | `go run ./demo stk 100 0712345678` followed by `go run ./demo query ws_CO_191220191020363925` | [`demo-stk-push.txt`](demo-stk-push.txt) |
| [`demo-callbacks.png`](demo-callbacks.png) | `go run ./demo callback demo/testdata/*.json` decoding an STK callback, a B2C result and a C2B confirmation | [`demo-callbacks.txt`](demo-callbacks.txt) |
| [`tests.png`](tests.png) | `go vet ./...` and `go test -race -cover ./...` | [`tests.txt`](tests.txt) |

Every value in the images comes from the program shown in the command line. The
demo commands were run against the mock Daraja API that ships with the repo
(`go run ./demo/mockdaraja`) with placeholder credentials, because the Safaricom
sandbox requires an approved app.

## Regenerating

The `*.txt` files are the untouched stdout/stderr of those commands.
`render_terminal.py` only adds the window frame and syntax colouring:

```sh
python3 -m pip install pillow
python3 docs/screenshots/render_terminal.py docs/screenshots/demo-stk-push.txt \
  "mpesa-api-go — go run ./demo stk" docs/screenshots/demo-stk-push.png
```

To reproduce the transcripts yourself:

```sh
go build -o /tmp/mockdaraja ./demo/mockdaraja
/tmp/mockdaraja -addr 127.0.0.1:8099 &

export MPESA_BASE_URL=http://127.0.0.1:8099
export MPESA_APP_KEY=demo-key MPESA_APP_SECRET=demo-secret
export MPESA_SHORT_CODE=174379
export MPESA_PASS_KEY=bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada1ed2c919

go run ./demo stk 100 0712345678
go run ./demo query ws_CO_191220191020363925
go run ./demo callback demo/testdata/stk-callback-success.json
```
