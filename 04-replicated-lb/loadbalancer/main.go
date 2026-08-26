package main

import (
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type loadBalancer struct {
	mu       sync.Mutex
	replicas []string
	next     int
}

func (lb *loadBalancer) getNextReplica() string {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	replica := lb.replicas[lb.next]
	lb.next = (lb.next + 1) % len(lb.replicas)

	return replica
}

func (lb *loadBalancer) proxyHandler(w http.ResponseWriter, r *http.Request) {
	replica := lb.getNextReplica()

	client := &http.Client{
		Timeout: time.Second * 2,
	}

	resp, err := client.Get(replica + r.URL.Path)
	if err != nil {
		http.Error(w, "could not get response", http.StatusInternalServerError)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "could not read the body", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(body)
}

func main() {
	lb := &loadBalancer{
		replicas: []string{"http://localhost:9001", "http://localhost:9002", "http://localhost:9003"},
	}

	http.HandleFunc("/", lb.proxyHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
