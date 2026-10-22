package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShipmentDecision(t *testing.T) {
	cases := []struct {
		name          string
		events        []shipmentEvent
		wantError     string
		wantDelivered bool
	}{
		{"proof required", []shipmentEvent{{ShipmentID: "S1", Kind: "delivered"}}, "proof_file", false},
		{"exception blocks delivery", []shipmentEvent{{ShipmentID: "S1", Kind: "exception"}, {ShipmentID: "S1", Kind: "delivered", ProofFile: "pod/S1.pdf"}}, "resolve exception", false},
		{"resolved exception permits delivery", []shipmentEvent{{ShipmentID: "S1", Kind: "exception"}, {ShipmentID: "S1", Kind: "in_transit"}, {ShipmentID: "S1", Kind: "delivered", ProofFile: "pod/S1.pdf"}}, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := &ledger{}
			var state shipment
			var err error
			for _, event := range tc.events {
				state, err = l.record(event)
			}
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil || state.Delivered != tc.wantDelivered {
				t.Fatalf("state = %+v, error = %v", state, err)
			}
		})
	}
}

func TestEnvelopeRejectionBeforeHTTPStatus(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/auth/phone/verify" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"INVALID_ARGUMENT","message":"invalid code"},"metadata":{}}`))
	}))
	defer upstream.Close()
	c := infraiClient{baseURL: upstream.URL, key: "test-key", http: upstream.Client()}
	_, err := c.verifyCode(context.Background(), "+15551234567", "000000")
	api, ok := err.(*apiError)
	if !ok || api.Code != "INVALID_ARGUMENT" || api.Status != 400 {
		t.Fatalf("error = %v", err)
	}
}
