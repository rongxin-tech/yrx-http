package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestClientJSONAndOptions(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		client := NewClient(ClientOptions{
			BaseURL: "https://example.test/v1", Timeout: time.Second,
			Headers: http.Header{"X-App": []string{"test"}}, BearerToken: "secret", RequestID: "req-1",
			HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != "https://example.test/v1/items" {
					t.Errorf("URL = %q", r.URL)
				}
				if r.Header.Get("X-App") != "test" || r.Header.Get("Authorization") != "Bearer secret" || r.Header.Get(RequestIDHeader) != "req-1" {
					t.Errorf("headers = %v", r.Header)
				}
				if r.Method != method {
					t.Errorf("method = %q, want %q", r.Method, method)
				}
				if method == http.MethodPost {
					var input map[string]string
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input["name"] != "item" {
						t.Errorf("input=%v err=%v", input, err)
					}
				}
				return response(http.StatusOK, `{"id":"1"}`), nil
			})},
		})
		var output map[string]string
		var input any
		if method == http.MethodPost {
			input = map[string]string{"name": "item"}
		}
		if err := client.DoJSON(context.Background(), method, "/items", input, &output, ""); err != nil {
			t.Fatal(err)
		}
		if output["id"] != "1" {
			t.Fatalf("output = %v", output)
		}
	}
}

func TestClientErrors(t *testing.T) {
	client := NewClient(ClientOptions{BaseURL: "https://example.test", HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/status" {
			return response(http.StatusTeapot, "no tea"), nil
		}
		return response(http.StatusOK, "not-json"), nil
	})}})
	err := client.Get(context.Background(), "/status", nil)
	var responseErr *ResponseError
	if !errors.As(err, &responseErr) || responseErr.StatusCode != http.StatusTeapot || string(responseErr.Body) != "no tea" {
		t.Fatalf("error = %#v", err)
	}
	err = client.Get(context.Background(), "/json", new(map[string]any))
	if !errors.As(err, &responseErr) || responseErr.Err == nil {
		t.Fatalf("invalid JSON error = %#v", err)
	}

	client = NewClient(ClientOptions{HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })}})
	err = client.Get(context.Background(), "https://example.test/", nil)
	if !errors.As(err, &responseErr) || responseErr.Err == nil {
		t.Fatalf("network error = %#v", err)
	}
}
