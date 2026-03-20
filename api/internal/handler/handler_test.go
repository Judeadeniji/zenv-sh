package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/Judeadeniji/zenv-sh/api/internal/testutil"
)

var ts *testutil.TestServer

func TestMain(m *testing.M) {
	srv, cleanup := testutil.SetupServerForMain()
	ts = srv
	code := m.Run()
	cleanup()
	os.Exit(code)
}

// doReq sends an HTTP request to the test server.
// token is used as a Bearer token in the Authorization header.
func doReq(t *testing.T, method, url string, body interface{}, token string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

// doReqWithCookie sends an HTTP request with both a Bearer token and a session cookie.
func doReqWithCookie(t *testing.T, method, url string, body interface{}, token string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		req.AddCookie(testutil.SessionCookie(token))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

// assertStatus fails the test if the response status code does not match.
func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("want status %d, got %d; body: %s", want, resp.StatusCode, string(body))
	}
}

// decodeJSON decodes the response body into dest.
func decodeJSON(t *testing.T, resp *http.Response, dest interface{}) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
}

// jsonBody is a convenience alias for map-based JSON request bodies.
type jsonBody = map[string]interface{}
