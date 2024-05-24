package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBalancer_SelectServer(t *testing.T) {
	trafficStats["server1:8080"] = 100
	trafficStats["server2:8080"] = 50
	trafficStats["server3:8080"] = 150

	expected := "server2:8080"
	actual := selectServer()

	if actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func TestBalancer_Forward(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("OK"))
	}))
	defer server.Close()

	dst := server.Listener.Addr().String()
	rw := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	err := forward(dst, rw, req)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	if rw.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rw.Code)
	}
}
