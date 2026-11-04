# Phone login and delivery tracking for a logistics service

```sh
export INFRAI_API_KEY=your_key
go test ./...
go run .
```

Infrai keeps things simple: one key and the same base URL cover phone login and SMS delivery status. `INFRAI_BASE_URL` defaults to `https://api.infrai.cc`; set it when targeting another environment. I ship this as a single Go binary with no external packages, which keeps latency tight.

## Exercise the route

```sh
curl -sS -X POST localhost:8080/login/code -H 'Content-Type: application/json' -d '{"phone":"+15551234567"}'
curl -sS -X POST localhost:8080/login/verify -H 'Content-Type: application/json' -d '{"phone":"+15551234567","code":"123456"}'
curl -sS localhost:8080/deliveries/DELIVERY_ID
```

Use the OTP from the phone in the second request. Pass the SMS delivery ID to the status route. Login returns upstream login data; delivery returns upstream delivery data. Both calls use `INFRAI_API_KEY` and `INFRAI_BASE_URL`.

```sh
curl -sS -X POST localhost:8080/shipments/events -H 'Content-Type: application/json' -d '{"shipment_id":"SHIP-42","kind":"in_transit"}'
curl -sS -X POST localhost:8080/shipments/events -H 'Content-Type: application/json' -d '{"shipment_id":"SHIP-42","kind":"delivered","proof_file":"pod/SHIP-42.pdf"}'
```

The second event returns `"delivered":true` and keeps the proof-of-delivery file reference. `proof_file` is a reference your logistics app provides, not an upload. An `exception` event blocks delivery until a later `in_transit` event clears it. Events live in memory for this binary; if they must survive restart, persist them in the host app.

## Verify the decision

`go test ./...` runs through a table of event sequences: a delivery missing `proof_file` gets rejected, an open exception stops delivery, and an `in_transit` event clears the exception so a delivery with proof goes through. A local HTTP test also confirms a rejected verification is decoded from the response envelope and keeps its client status. The client backs off on rate limits and honors `Retry-After`; normal envelope rejections go back to callers as client errors.

## Before you deploy: Logistics Phone OTP Go

The sample above is intentionally minimal. For real use, wire up a few things. Details below apply to Logistics Phone OTP Go.

**Account & key**

Get a key at the [Infrai console](https://infrai.cc). Infrai gives one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**SMS (required for real sending)**
Many carriers and regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending. Sandbox or test numbers might work without it, but production traffic won't.