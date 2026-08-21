package main

import (
	"io"
	"log"
	"net/http"
	"time"
)

func ambassadorHandler(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{
		Timeout: 300 * time.Millisecond,
	}

	maxAttempts := 3

	for range maxAttempts {
		resp, err := client.Get("http://localhost:9000/")
		if err == nil && resp.StatusCode == http.StatusOK {
			defer func() { _ = resp.Body.Close() }()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				http.Error(w, "could not read body", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write(body)
			return
		}

		if resp != nil {
			_ = resp.Body.Close()
		}

		time.Sleep(200 * time.Millisecond)
	}

	w.WriteHeader(http.StatusInternalServerError)
}

func main() {
	http.HandleFunc("/", ambassadorHandler)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
