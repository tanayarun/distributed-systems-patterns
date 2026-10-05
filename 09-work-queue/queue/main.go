package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
)

type task struct {
	ID      int
	Payload string
	Status  string
	Result  string
}

type queue struct {
	mu      sync.Mutex
	nextID  int
	tasks   map[int]*task
	pending []int
}

func (q *queue) submitHandler(w http.ResponseWriter, r *http.Request) {
	payload := r.URL.Query().Get("payload")
	if payload == "" {
		http.Error(w, "payload required", http.StatusBadRequest)
		return
	}

	q.mu.Lock()
	q.nextID++
	id := q.nextID
	q.tasks[id] = &task{ID: id, Payload: payload, Status: "pending"}
	q.pending = append(q.pending, id)
	q.mu.Unlock()

	_, _ = fmt.Fprintf(w, "%d\n", id)
}

func (q *queue) pullHandler(w http.ResponseWriter, r *http.Request) {
	q.mu.Lock()
	if len(q.pending) == 0 {
		q.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	id := q.pending[0]
	q.pending = q.pending[1:]
	t := q.tasks[id]
	t.Status = "running"
	snapshot := *t
	q.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snapshot)
}

func (q *queue) completeHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	result := r.URL.Query().Get("result")

	q.mu.Lock()
	defer q.mu.Unlock()

	t, exists := q.tasks[id]
	if !exists {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if t.Status != "running" {
		http.Error(w, "task is not running", http.StatusConflict)
		return
	}
	t.Status = "done"
	t.Result = result
	w.WriteHeader(http.StatusOK)
}

func (q *queue) statusHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	q.mu.Lock()
	t, exists := q.tasks[id]
	if !exists {
		q.mu.Unlock()
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	snapshot := *t
	q.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snapshot)
}

func main() {
	q := &queue{tasks: make(map[int]*task)}
	http.HandleFunc("/submit", q.submitHandler)
	http.HandleFunc("/pull", q.pullHandler)
	http.HandleFunc("/complete", q.completeHandler)
	http.HandleFunc("/status", q.statusHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
