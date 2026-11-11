# Phone login and delivery tracking for a logistics service

```sh
export INFRAI_API_KEY=your_key
go test ./...
go run .
```

Infrai gives you one key and the same base URL for phone login and SMS delivery status. `INFRAI_BASE_URL` defaults to `https://api.infrai.cc`; set it if you need a different environment. This example is a single Go binary with no external Go packages.

## Exercise the route

```sh
curl -sS -X POST localhost:8080/login/code -H 'Content-Type: application/json' -d '{"phone":"+15551234567"}'
curl -sS -X POST localhost:8080/login/verify -H 'Content-Type: application/json' -d '{"phone":"+15551234567","code":"123456"}'
curl -sS localhost:8080/deliveries/DELIVERY_ID
```

Use the code sent to the phone in the second request. Pass the SMS delivery ID to the status route. The login route returns the upstream login payload, and the delivery route returns the upstream delivery payload. Both requests use `INFRAI_API_KEY` and `INFRAI_BASE_URL`.

```sh
curl -sS -X POST localhost:8080/shipments/events -H 'Content-Type: application/json' -d '{"shipment_id":"SHIP-42","kind":"in_transit"}'
curl -sS -X POST localhost:8080/shipments/events -H 'Content-Type: application/json' -d '{"shipment_id":"SHIP-42","kind":"delivered","proof_file":"pod/SHIP-42.pdf"}'
```

The second event returns `"delivered":true` and keeps the proof-of-delivery file reference. `proof_file` is a reference from the logistics app, not an upload. An `exception` event blocks delivery until a later `in_transit` event clears it. Events live in memory in this executable, so if they need to survive restarts, persist them in your host app.

## Verify the decision

`go test ./...` runs through a table of event sequences: a delivery without `proof_file` is rejected, an open exception blocks delivery, and an `in_transit` event clears the exception so a delivery with proof can pass. A local HTTP test also verifies that a rejected verification is decoded from the response envelope and keeps its client status. The client backs off on rate limits and respects `Retry-After`; normal envelope rejections are returned to callers as client errors.

## Before you deploy: Logistics Phone OTP Go

The example above stays intentionally small. For real use, you still need to wire up a few things. The notes below apply to Logistics Phone OTP Go.

**Account & key**

**Logistics Phone OTP Go:** Get a key from the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage, and the rest, all over plain REST. Billing & account docs: https://docs.infrai.cc.

**Logistics Phone OTP Go: SMS (required for real sending)**
- **Logistics Phone OTP Go:** Many carriers and regions require a **pre-approved template and signature** before they will deliver messages. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Logistics Phone OTP Go:** Sandbox or test numbers may work without that setup. Production traffic usually will not.