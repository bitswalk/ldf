package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// ListOptions Tests
// =============================================================================

func TestListOptions_QueryString_Nil(t *testing.T) {
	var opts *ListOptions
	if qs := opts.QueryString(); qs != "" {
		t.Errorf("expected empty string for nil opts, got %q", qs)
	}
}

func TestListOptions_QueryString_Empty(t *testing.T) {
	opts := &ListOptions{}
	if qs := opts.QueryString(); qs != "" {
		t.Errorf("expected empty string for zero-value opts, got %q", qs)
	}
}

func TestListOptions_QueryString_LimitOnly(t *testing.T) {
	opts := &ListOptions{Limit: 10}
	qs := opts.QueryString()
	if qs != "?limit=10" {
		t.Errorf("expected ?limit=10, got %q", qs)
	}
}

func TestListOptions_QueryString_AllFields(t *testing.T) {
	opts := &ListOptions{
		Limit:       50,
		Offset:      20,
		Status:      "ready",
		VersionType: "stable",
		Category:    "kernel",
	}
	qs := opts.QueryString()
	if !strings.HasPrefix(qs, "?") {
		t.Fatalf("expected query string to start with ?, got %q", qs)
	}
	for _, expected := range []string{"limit=50", "offset=20", "status=ready", "version_type=stable", "category=kernel"} {
		if !strings.Contains(qs, expected) {
			t.Errorf("query string %q missing %q", qs, expected)
		}
	}
}

func TestListOptions_QueryString_SkipsZeroValues(t *testing.T) {
	opts := &ListOptions{Status: "pending"}
	qs := opts.QueryString()
	if strings.Contains(qs, "limit") {
		t.Errorf("zero limit should be omitted, got %q", qs)
	}
	if strings.Contains(qs, "offset") {
		t.Errorf("zero offset should be omitted, got %q", qs)
	}
	if !strings.Contains(qs, "status=pending") {
		t.Errorf("expected status=pending in %q", qs)
	}
}

// =============================================================================
// APIError Tests
// =============================================================================

func TestAPIError_Error_WithCode(t *testing.T) {
	err := &APIError{StatusCode: 401, ErrorCode: "unauthorized", Message: "invalid token"}
	s := err.Error()
	if !strings.Contains(s, "unauthorized") {
		t.Errorf("expected error code in message, got %q", s)
	}
	if !strings.Contains(s, "invalid token") {
		t.Errorf("expected message in error, got %q", s)
	}
	if !strings.Contains(s, "401") {
		t.Errorf("expected status code in error, got %q", s)
	}
}

func TestAPIError_Error_WithoutCode(t *testing.T) {
	err := &APIError{StatusCode: 500, Message: "internal error"}
	s := err.Error()
	if !strings.Contains(s, "500") || !strings.Contains(s, "internal error") {
		t.Errorf("unexpected error format: %q", s)
	}
}

func TestAPIError_Hint401(t *testing.T) {
	err := &APIError{StatusCode: 401, Message: "unauthorized"}
	if !strings.Contains(err.Error(), "ldfctl login") {
		t.Error("expected login hint for 401")
	}
}

func TestAPIError_Hint403(t *testing.T) {
	err := &APIError{StatusCode: 403, Message: "forbidden"}
	if !strings.Contains(err.Error(), "Permission denied") {
		t.Error("expected permission hint for 403")
	}
}

func TestAPIError_Hint404(t *testing.T) {
	err := &APIError{StatusCode: 404, Message: "not found"}
	if !strings.Contains(err.Error(), "not found") {
		t.Error("expected not found hint for 404")
	}
}

func TestAPIError_Hint409(t *testing.T) {
	err := &APIError{StatusCode: 409, Message: "conflict"}
	if !strings.Contains(err.Error(), "already exists") {
		t.Error("expected conflict hint for 409")
	}
}

func TestAPIError_NoHint200(t *testing.T) {
	err := &APIError{StatusCode: 500, Message: "server error"}
	if strings.Contains(err.Error(), "Hint:") {
		t.Error("unexpected hint for 500")
	}
}

// =============================================================================
// Client Tests
// =============================================================================

func TestNew(t *testing.T) {
	c := New("http://localhost:8443")
	if c.BaseURL != "http://localhost:8443" {
		t.Errorf("expected base URL http://localhost:8443, got %s", c.BaseURL)
	}
	if c.HTTPClient == nil {
		t.Error("expected non-nil HTTP client")
	}
	if c.Token != "" {
		t.Error("expected empty token on new client")
	}
}

func TestClient_Get_Success(t *testing.T) {
	expected := map[string]string{"message": "ok"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/test" {
			t.Errorf("expected path /v1/test, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := New(srv.URL)
	var result map[string]string
	if err := c.Get(context.Background(), "/v1/test", &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["message"] != "ok" {
		t.Errorf("expected message=ok, got %v", result)
	}
}

func TestClient_Get_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "not_found", Message: "resource not found"})
	}))
	defer srv.Close()

	c := New(srv.URL)
	var result interface{}
	err := c.Get(context.Background(), "/v1/missing", &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}

func TestClient_Post_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "123", "name": body["name"]})
	}))
	defer srv.Close()

	c := New(srv.URL)
	req := map[string]string{"name": "test"}
	var result map[string]string
	if err := c.Post(context.Background(), "/v1/items", req, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["name"] != "test" {
		t.Errorf("expected name=test, got %v", result)
	}
}

func TestClient_Delete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL)
	if err := c.Delete(context.Background(), "/v1/items/123", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_AuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Authorization: Bearer test-token, got %s", auth)
		}
		token := r.Header.Get("X-Subject-Token")
		if token != "test-token" {
			t.Errorf("expected X-Subject-Token: test-token, got %s", token)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.Token = "test-token"
	c.Get(context.Background(), "/v1/test", nil)
}

func TestClient_Put_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"updated": "true"})
	}))
	defer srv.Close()

	c := New(srv.URL)
	var result map[string]string
	if err := c.Put(context.Background(), "/v1/items/123", map[string]string{"name": "updated"}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["updated"] != "true" {
		t.Errorf("expected updated=true, got %v", result)
	}
}

func TestClient_ErrorResponse_Structured(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "bad_request", Message: "invalid input"})
	}))
	defer srv.Close()

	c := New(srv.URL)
	err := c.Get(context.Background(), "/v1/test", nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.ErrorCode != "bad_request" {
		t.Errorf("expected error code bad_request, got %s", apiErr.ErrorCode)
	}
	if apiErr.Message != "invalid input" {
		t.Errorf("expected message 'invalid input', got %s", apiErr.Message)
	}
}

func TestClient_ErrorResponse_PlainText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("something went wrong"))
	}))
	defer srv.Close()

	c := New(srv.URL)
	err := c.Get(context.Background(), "/v1/test", nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("expected 500, got %d", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Message, "something went wrong") {
		t.Errorf("expected plain text message, got %s", apiErr.Message)
	}
}
