package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
	"tailscale.com/util/set"
)

func admit(t *testing.T, h http.Handler, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAdmissionHandler(t *testing.T) {
	allowed, denied := key.NewNode().Public(), key.NewNode().Public()
	c := newTestCache()
	c.update("t", set.Of(allowed))
	h := newHandler(c, slog.New(slog.NewTextHandler(io.Discard, nil)))

	body := func(k key.NodePublic) string {
		b, err := json.Marshal(tailcfg.DERPAdmitClientRequest{NodePublic: k})
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	tests := []struct {
		name, method, body string
		wantStatus         int
		wantAllow          bool
	}{
		{"allowed", http.MethodPost, body(allowed), http.StatusOK, true},
		{"denied", http.MethodPost, body(denied), http.StatusOK, false},
		{"wrong method", http.MethodGet, "", http.StatusMethodNotAllowed, false},
		{"bad json", http.MethodPost, "{", http.StatusBadRequest, false},
		{"trailing data", http.MethodPost, body(allowed) + body(allowed), http.StatusBadRequest, false},
		{"too large", http.MethodPost, `{"x":"` + strings.Repeat("a", maxRequestBodyBytes) + `"}`, http.StatusBadRequest, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := admit(t, h, tc.method, tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				return
			}
			var resp tailcfg.DERPAdmitClientResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatal(err)
			}
			if resp.Allow != tc.wantAllow {
				t.Fatalf("allow = %v, want %v", resp.Allow, tc.wantAllow)
			}
		})
	}
}
