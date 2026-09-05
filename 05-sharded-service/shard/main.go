package main

import (
	"flag"
	"log"
	"net/http"
	"sync"
)

type store struct {
	mu   sync.Mutex
	data map[string]string
}

var port = flag.String("port", ":8080", "port to listen on")

func (s *store) getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")

	s.mu.Lock()
	value, exists := s.data[key]
	s.mu.Unlock()

	if !exists {
		http.Error(w, "could not find key", http.StatusNotFound)
		return
	}

	_, _ = w.Write([]byte(value))
}

func (s *store) setHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "key not found", http.StatusBadRequest)
		return
	}

	value := r.URL.Query().Get("value")

	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func main() {
	flag.Parse()
	s := &store{data: make(map[string]string)}

	http.HandleFunc("/get", s.getHandler)
	http.HandleFunc("/set", s.setHandler)
	log.Fatal(http.ListenAndServe(*port, nil))
}
