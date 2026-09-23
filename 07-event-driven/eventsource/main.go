package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type EventHandler func(payload string) string

func handleUserCreated(payload string) string {
	return fmt.Sprintf("welcome email queued for payload: %s", payload)
}

func handleFileUploaded(payload string) string {
	return fmt.Sprintf("resize job started for file: %s", payload)
}

func handleOrderPlaced(payload string) string {
	return fmt.Sprintf("order confirmation sent for: %s", payload)
}

type event struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type eventSource struct {
	handlers map[string]EventHandler
}

func (es *eventSource) eventHandler(w http.ResponseWriter, r *http.Request) {
	var ev event
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		http.Error(w, "invalid event body", http.StatusBadRequest)
		return
	}

	handler, exists := es.handlers[ev.Type]
	if !exists {
		http.Error(w, fmt.Sprintf("no handler registered for event type %q", ev.Type), http.StatusNotFound)
		return
	}

	start := time.Now()
	result := handler(ev.Payload)
	duration := time.Since(start)

	_, _ = fmt.Fprintf(w, "event: %s\nresult: %s\nhandled in: %v\n", ev.Type, result, duration)
}

func main() {
	es := &eventSource{
		handlers: map[string]EventHandler{
			"user.created":  handleUserCreated,
			"file.uploaded": handleFileUploaded,
			"order.placed":  handleOrderPlaced,
		},
	}

	http.HandleFunc("/events", es.eventHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
