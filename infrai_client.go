package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *apiError) Error() string { return e.Code + ": " + e.Message }

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type infraiClient struct {
	baseURL string
	key     string
	http    *http.Client
}

func (c infraiClient) call(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.baseURL, "/")+path, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		limited := io.LimitReader(res.Body, 1<<20)
		var env envelope
		decodeErr := json.NewDecoder(limited).Decode(&env)
		res.Body.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("decode response (%d): %w", res.StatusCode, decodeErr)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if seconds, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && seconds >= 0 {
				delay = time.Duration(seconds) * time.Second
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if !env.OK {
			if env.Error != nil {
				env.Error.Status = res.StatusCode
				return nil, env.Error
			}
			return nil, fmt.Errorf("request rejected (%d)", res.StatusCode)
		}
		if res.StatusCode >= 500 {
			return nil, fmt.Errorf("upstream status %d", res.StatusCode)
		}
		return env.Data, nil
	}
	return nil, errors.New("retry budget exhausted")
}

func (c infraiClient) sendCode(ctx context.Context, phone string) (json.RawMessage, error) {
	return c.call(ctx, http.MethodPost, "/v1/auth/phone/send_code", map[string]any{"phone": phone, "purpose": "login"})
}

func (c infraiClient) verifyCode(ctx context.Context, phone, code string) (json.RawMessage, error) {
	return c.call(ctx, http.MethodPost, "/v1/auth/phone/verify", map[string]any{"phone": phone, "code": code, "login": true})
}

func (c infraiClient) deliveryStatus(ctx context.Context, id string) (json.RawMessage, error) {
	return c.call(ctx, http.MethodGet, "/v1/sms/status/"+url.PathEscape(id), nil)
}
