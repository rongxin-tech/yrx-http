package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSONAndError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, http.StatusCreated, map[string]string{"ok": "yes"})
	if w.Code != http.StatusCreated || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("status/header = %d/%q", w.Code, w.Header().Get("Content-Type"))
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body["ok"] != "yes" {
		t.Fatalf("body = %s, err = %v", w.Body, err)
	}

	w = httptest.NewRecorder()
	WriteHTTPError(w, NewError(http.StatusBadRequest, "bad input"))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `"error":"bad input"`) {
		t.Fatalf("error response = %d %s", w.Code, w.Body)
	}
}

func TestRequireMethod(t *testing.T) {
	w := httptest.NewRecorder()
	if RequireMethod(w, httptest.NewRequest(http.MethodGet, "/", nil), http.MethodPost) {
		t.Fatal("GET unexpectedly accepted")
	}
	if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("response = %d, Allow=%q", w.Code, w.Header().Get("Allow"))
	}
}

func TestParseBearerToken(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		ok          bool
	}{
		{"Bearer abc", "abc", true}, {"bEaReR abc", "abc", true}, {"", "", false}, {"Basic abc", "", false}, {"Bearer", "", false}, {"Bearer a b", "", false},
	} {
		got, err := ParseBearerToken(tc.input)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("ParseBearerToken(%q) = %q, %v", tc.input, got, err)
		}
	}
}

func TestWithRequestLogging(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := WithRequestLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestID(r) == "" {
			t.Error("request ID missing from context")
		}
		w.WriteHeader(http.StatusAccepted)
	}), logger)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if w.Header().Get(RequestIDHeader) == "" || w.Code != http.StatusAccepted {
		t.Fatalf("headers/status = %v/%d", w.Header(), w.Code)
	}
	for _, field := range []string{`"method":"GET"`, `"path":"/healthz"`, `"status":202`, `"duration"`, `"request_id"`} {
		if !strings.Contains(logs.String(), field) {
			t.Errorf("log missing %s: %s", field, logs.String())
		}
	}
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "upstream-id")
	handler.ServeHTTP(w, req)
	if w.Header().Get(RequestIDHeader) != "upstream-id" {
		t.Fatalf("request ID = %q", w.Header().Get(RequestIDHeader))
	}
}
