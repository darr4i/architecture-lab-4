package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForward(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("OK"))
	}))
	defer backend.Close()

	// Инициализируем карту servers для теста
	servers = map[string]*serverStatus{
		backend.URL[len("http://"):]: {
			address:  backend.URL[len("http://"):],
			traffic:  0,
			isHealthy: true,
		},
	}

	rw := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://example.com", nil)

	err := forward(backend.URL[len("http://"):], rw, req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if rw.Code != http.StatusOK {
		t.Errorf("expected status %v, got %v", http.StatusOK, rw.Code)
	}
	if rw.Body.String() != "OK" {
		t.Errorf("expected body %v, got %v", "OK", rw.Body.String())
	}
}

func TestGetHealthyServer(t *testing.T) {
	servers = map[string]*serverStatus{
		"server1": {address: "server1", traffic: 10, isHealthy: true},
		"server2": {address: "server2", traffic: 5, isHealthy: true},
		"server3": {address: "server3", traffic: 15, isHealthy: false},
	}
	server := getHealthyServer()
	if server != "server2" {
		t.Errorf("expected server2, got %v", server)
	}
}
