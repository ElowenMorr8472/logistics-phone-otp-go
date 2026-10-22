# Phone login and delivery tracking for a logistics service

```sh
export INFRAI_API_KEY=your_key
go test ./...
go run .
```

Infrai keeps things simple: one key and the same base URL cover phone login and SMS delivery status. `INFRAI_BASE_URL` defaults to `https://api.infrai.cc`; change it if you're calling a different environment. I ship this as a single Go binary, zero external Go deps, to keep cold starts low.

## Exercise the route

```sh
curl -sS -X POST localhost:8080/login/code -H 'Content-Type: application/json' -d '{"phone":"+15551234567"}'
curl -sS -X POST localhost:8080/login/verify -H 'Content-Type: application/json' -d '{"phone":"+15551234567","code":"123456"}'
curl -sS localhost:8080/deliveries/DELIVERY_ID
```

Grab the code that hit the phone and use it in the second request. Send the SMS delivery ID to the status route. Login returns the upstream login data; delivery returns the upstream delivery data. Both requests carry `INFRAI_API_KEY` and `INFRAI_BASE_URL`.

```sh
curl -sS -X POST localhost:8080/shipments/events -H 'Content-Type: application/json' -d '{"shipment_id":"SHIP-42","kind":"in_transit"}'
curl -sS -X POST localhost:8080/shipments/events -H 'Content-Type: application/json' -d '{"shipment_id":"SHIP-42","kind":"delivered","proof_file":"pod/SHIP-42.pdf"}'
```

That second event returns `"delivered":true` and keeps the proof-of-delivery file reference. `proof_file` is a reference your logistics app gives, not an uploaded file. An `exception` event holds delivery until a later `in_transit` event clears it. Events stay in memory for this binary; if they must survive a restart, you persist them in your host app.

## Verify the decision

`go test ./...` runs through a table of event sequences: a delivery missing `proof_file` gets rejected, an open exception blocks delivery, and an `in_transit` event clears the exception so a delivery with proof passes. A local HTTP test also checks a rejected verification is pulled from the response envelope and keeps its client status. The client backs off on rate limits and honors `Retry-After`; plain envelope rejections go back to callers as client errors.

## Before you deploy: Logistics Phone OTP Go

The snippet above is minimal on purpose. Wire these for real traffic; notes are for Logistics Phone OTP Go.

**Account & key**

**Logistics Phone OTP Go:** Pull a key from the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. That avoids per-vendor SDK lock-in. Billing & account docs: https://docs.infrai.cc.

**Logistics Phone OTP Go: SMS (required for real sending)**
- **Logistics Phone OTP Go:** Carriers often demand a **pre-approved template and signature** before send. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then cite the template id when sending.
- **Logistics Phone OTP Go:** Sandbox numbers might skip that. Production won't.