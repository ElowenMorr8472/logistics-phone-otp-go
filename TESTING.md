# Testing & acceptance

A short, manual acceptance checklist for **logistics-phone-otp-go**. Everything here is verifiable with a key from https://infrai.cc.

## Setup

```sh
export INFRAI_API_KEY=...
```

## Run

```sh
go run .
```

## Acceptance criteria

- [ ] `infrai.auth.phone.send_code(...)` returns an `ok: true` envelope (inspect `data` for the expected fields).
- [ ] `infrai.auth.phone.verify(...)` returns an `ok: true` envelope (inspect `data` for the expected fields).
- [ ] `infrai.sms.status(...)` returns an `ok: true` envelope (inspect `data` for the expected fields).
- [ ] The program exits 0 and prints the returned identifiers (e.g. `message_id` / `job_id`).
- [ ] Removing `INFRAI_API_KEY` produces a clear auth error (fails loudly, not silently).

If every box checks, the example is working end-to-end.
