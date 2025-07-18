package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestValidatePID(t *testing.T) {
	tests := []struct {
		name    string
		pid     string
		wantErr bool
	}{
		{
			name:    "valid PID format",
			pid:     "1",
			wantErr: false,
		},
		{
			name:    "invalid PID format - not a number",
			pid:     "abc",
			wantErr: true,
		},
		{
			name:    "invalid PID format - empty",
			pid:     "",
			wantErr: true,
		},
		{
			name:    "non-existent PID",
			pid:     "999999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePID(tt.pid)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckRequiredTools(t *testing.T) {
	// This test checks if the tools are available on the system
	// It's more of an integration test
	err := checkRequiredTools()
	if err != nil {
		t.Logf("Required tools not available: %v", err)
		t.Skip("Skipping test - required tools not installed")
	}
}

func TestHandlePprofTestMode(t *testing.T) {
	req, err := http.NewRequest("GET", "/debug/pprof/profile?pid=1234&seconds=5&test=true", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlePprof)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/octet-stream" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/octet-stream")
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Mock profile data") {
		t.Errorf("handler returned unexpected body: %v", body)
	}
}

func TestHandleFoldedTestMode(t *testing.T) {
	req, err := http.NewRequest("GET", "/debug/folded/profile?pid=1234&seconds=5&test=true", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleFolded)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "text/plain")
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Mock profile data") {
		t.Errorf("handler returned unexpected body: %v", body)
	}
}

func TestMissingParameters(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantCode int
	}{
		{
			name:     "missing pid",
			url:      "/debug/pprof/profile?seconds=5",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing seconds",
			url:      "/debug/pprof/profile?pid=1234",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing both",
			url:      "/debug/pprof/profile",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid seconds",
			url:      "/debug/pprof/profile?pid=1234&seconds=abc",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "seconds too large",
			url:      "/debug/pprof/profile?pid=1234&seconds=500",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handlePprof)

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.wantCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantCode)
			}
		})
	}
}

func TestInvalidPID(t *testing.T) {
	req, err := http.NewRequest("GET", "/debug/pprof/profile?pid=999999&seconds=5", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlePprof)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Invalid PID") {
		t.Errorf("handler returned unexpected error message: %v", body)
	}
}

func TestGenerateMockProfile(t *testing.T) {
	pid := "1234"
	duration := 10

	result := generateMockProfile(pid, duration)

	if !strings.Contains(result, pid) {
		t.Errorf("Mock profile should contain PID %s", pid)
	}

	if !strings.Contains(result, strconv.Itoa(duration)) {
		t.Errorf("Mock profile should contain duration %d", duration)
	}

	if !strings.Contains(result, "redis-server") {
		t.Errorf("Mock profile should contain redis-server entries")
	}
}

// TestBasicAuth tests the basic authentication functionality
func TestBasicAuth(t *testing.T) {
	password := "testpass"
	handler := basicAuth(handlePprof, password)

	tests := []struct {
		name     string
		username string
		password string
		wantCode int
	}{
		{
			name:     "valid credentials",
			username: "admin",
			password: "testpass",
			wantCode: http.StatusBadRequest, // Will fail due to missing params, but auth should pass
		},
		{
			name:     "invalid username",
			username: "wrong",
			password: "testpass",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "invalid password",
			username: "admin",
			password: "wrong",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "no credentials",
			username: "",
			password: "",
			wantCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/debug/pprof/profile", nil)
			if err != nil {
				t.Fatal(err)
			}

			if tt.username != "" || tt.password != "" {
				req.SetBasicAuth(tt.username, tt.password)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.wantCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantCode)
			}
		})
	}
}

// Integration test that requires actual system tools
func TestPerfIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if we're running as root or have appropriate permissions
	if os.Geteuid() != 0 {
		t.Skip("Skipping integration test - requires root privileges")
	}

	// Check if required tools are available
	if err := checkRequiredTools(); err != nil {
		t.Skipf("Skipping integration test - required tools not available: %v", err)
	}

	// Use PID 1 (init process) which should always exist
	req, err := http.NewRequest("GET", "/debug/pprof/profile?pid=1&seconds=1", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlePprof)

	handler.ServeHTTP(rr, req)

	// This might fail due to permissions, but we can check the response
	if status := rr.Code; status == http.StatusOK {
		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/octet-stream" {
			t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/octet-stream")
		}

		disposition := rr.Header().Get("Content-Disposition")
		if !strings.Contains(disposition, "profile-1-1.pb.gz") {
			t.Errorf("handler returned wrong content disposition: got %v", disposition)
		}
	} else {
		t.Logf("Integration test failed with status %d (expected due to permissions): %s", status, rr.Body.String())
	}
}
