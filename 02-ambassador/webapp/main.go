package main

import (
	"io"
	"log"
	"net/http"
	"time"
)

func sampleHandler(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	resp, err := client.Get("http://localhost:8081/")
	if err != nil {
		http.Error(w, "Backend failed", http.StatusInternalServerError)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("could not read body"))
		return
	}

	_, _ = w.Write(body)
}

func main() {
	http.HandleFunc("/", sampleHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
