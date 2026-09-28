package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

const leaseDuration = 5 * time.Second

type lease struct {
	mu        sync.Mutex
	holder    string
	expiresAt time.Time
}

func (l *lease) isExpired() bool {
	return time.Now().After(l.expiresAt)
}

func (l *lease) acquireHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.holder != "" && l.holder != id && !l.isExpired() {
		http.Error(w, fmt.Sprintf("lease held by %s until %v", l.holder, l.expiresAt), http.StatusConflict)
		return
	}

	l.holder = id
	l.expiresAt = time.Now().Add(leaseDuration)
	_, _ = fmt.Fprintf(w, "acquired by %s, expires at %v\n", l.holder, l.expiresAt)
}

func (l *lease) renewHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.holder != id || l.isExpired() {
		http.Error(w, "not the current holder", http.StatusConflict)
		return
	}

	l.expiresAt = time.Now().Add(leaseDuration)
	_, _ = fmt.Fprintf(w, "renewed by %s, expires at %v\n", l.holder, l.expiresAt)
}

func (l *lease) statusHandler(w http.ResponseWriter, r *http.Request) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.holder == "" || l.isExpired() {
		_, _ = fmt.Fprintf(w, "no active leader\n")
		return
	}
	_, _ = fmt.Fprintf(w, "leader: %s, expires at %v\n", l.holder, l.expiresAt)
}

func main() {
	l := &lease{}
	http.HandleFunc("/acquire", l.acquireHandler)
	http.HandleFunc("/renew", l.renewHandler)
	http.HandleFunc("/status", l.statusHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
