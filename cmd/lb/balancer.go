package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/roman-mazur/architecture-practice-4-template/httptools"
	"github.com/roman-mazur/architecture-practice-4-template/signal"
)

var (
	port        = flag.Int("port", 8090, "load balancer port")
	timeoutSec  = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https       = flag.Bool("https", false, "whether backends support HTTPs")
	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

var (
	timeout     = time.Duration(*timeoutSec) * time.Second
	serversPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}
)

type serverStatus struct {
	address  string
	traffic  int64
	isHealthy bool
}

var (
	mu       sync.Mutex
	servers  = make(map[string]*serverStatus)
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func forward(dst string, rw http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		mu.Lock()
		servers[dst].traffic += resp.ContentLength
		mu.Unlock()
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

func getHealthyServer() string {
	mu.Lock()
	defer mu.Unlock()
	var bestServer *serverStatus
	for _, server := range servers {
		if server.isHealthy {
			if bestServer == nil || server.traffic < bestServer.traffic {
				bestServer = server
			}
		}
	}
	if bestServer != nil {
		return bestServer.address
	}
	return ""
}

func main() {
	flag.Parse()

	for _, server := range serversPool {
		servers[server] = &serverStatus{
			address:  server,
			traffic:  0,
			isHealthy: false,
		}
		go func(server string) {
			for range time.Tick(10 * time.Second) {
				servers[server].isHealthy = health(server)
			}
		}(server)
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		server := getHealthyServer()
		if server == "" {
			http.Error(rw, "No healthy servers available", http.StatusServiceUnavailable)
			return
		}
		if err := forward(server, rw, r); err != nil {
			http.Error(rw, "Failed to forward request", http.StatusServiceUnavailable)
		}
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
