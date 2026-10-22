package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
)

type shipmentEvent struct {
	ShipmentID string `json:"shipment_id"`
	Kind       string `json:"kind"`
	Note       string `json:"note,omitempty"`
	ProofFile  string `json:"proof_file,omitempty"`
}

type shipment struct {
	ID        string          `json:"id"`
	Events    []shipmentEvent `json:"events"`
	Exception bool            `json:"exception"`
	Delivered bool            `json:"delivered"`
}

type ledger struct {
	mu    sync.Mutex
	items map[string]shipment
}

func (l *ledger) record(e shipmentEvent) (shipment, error) {
	if e.ShipmentID == "" {
		return shipment{}, errors.New("shipment_id is required")
	}
	if e.Kind != "in_transit" && e.Kind != "exception" && e.Kind != "delivered" {
		return shipment{}, errors.New("invalid event kind")
	}
	if e.Kind == "delivered" && e.ProofFile == "" {
		return shipment{}, errors.New("delivery needs proof_file")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.items == nil {
		l.items = make(map[string]shipment)
	}
	s := l.items[e.ShipmentID]
	s.ID = e.ShipmentID
	if s.Delivered {
		return shipment{}, errors.New("shipment already delivered")
	}
	if e.Kind == "exception" {
		s.Exception = true
	}
	if e.Kind == "in_transit" {
		s.Exception = false
	}
	if e.Kind == "delivered" {
		if s.Exception {
			return shipment{}, errors.New("resolve exception before delivery")
		}
		s.Delivered = true
	}
	s.Events = append(s.Events, e)
	l.items[e.ShipmentID] = s
	return s, nil
}

type service struct {
	client infraiClient
	ledger *ledger
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func failure(w http.ResponseWriter, err error) {
	var api *apiError
	if errors.As(err, &api) {
		status := http.StatusBadGateway
		if api.Status >= 400 && api.Status < 500 {
			status = api.Status
		}
		writeJSON(w, status, map[string]any{"error": api})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
}

func (s service) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/code", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Phone string `json:"phone"`
		}
		if json.NewDecoder(r.Body).Decode(&input) != nil || input.Phone == "" {
			failure(w, errors.New("phone is required"))
			return
		}
		data, err := s.client.sendCode(r.Context(), input.Phone)
		if err != nil {
			failure(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"delivery": data})
	})
	mux.HandleFunc("POST /login/verify", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Phone string `json:"phone"`
			Code  string `json:"code"`
		}
		if json.NewDecoder(r.Body).Decode(&input) != nil || input.Phone == "" || input.Code == "" {
			failure(w, errors.New("phone and code are required"))
			return
		}
		data, err := s.client.verifyCode(r.Context(), input.Phone, input.Code)
		if err != nil {
			failure(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"login": data})
	})
	mux.HandleFunc("GET /deliveries/{id}", func(w http.ResponseWriter, r *http.Request) {
		data, err := s.client.deliveryStatus(r.Context(), r.PathValue("id"))
		if err != nil {
			failure(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"delivery": data})
	})
	mux.HandleFunc("POST /shipments/events", func(w http.ResponseWriter, r *http.Request) {
		var input shipmentEvent
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			failure(w, errors.New("invalid event"))
			return
		}
		input.ShipmentID = strings.TrimSpace(input.ShipmentID)
		state, err := s.ledger.record(input)
		if err != nil {
			failure(w, err)
			return
		}
		writeJSON(w, http.StatusOK, state)
	})
	return mux
}
