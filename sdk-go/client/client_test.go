package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestClient_Requests(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/get", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Missing or invalid token")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/v1/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "value" {
			t.Errorf("Invalid POST body")
		}
		w.WriteHeader(http.StatusCreated)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "test-token")

	// Test GET
	body, code, err := c.get("/v1/get", url.Values{"q": {"search"}})
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	if code != http.StatusOK {
		t.Errorf("Expected 200, got %d", code)
	}
	if string(body) != `{"status":"ok"}` {
		t.Errorf("Unexpected body: %s", string(body))
	}

	// Test POST
	_, code, err = c.post("/v1/post", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	if code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", code)
	}
}

func TestClient_RequestError(t *testing.T) {
	c := New("http://invalid-url-that-does-not-exist.local", "token")
	_, _, err := c.get("/v1/test", nil)
	if err == nil {
		t.Errorf("Expected error for unreachable host")
	}
}
