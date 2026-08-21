package main

import (
	"log"
	"math/rand/v2"
	"net/http"
	"time"
)

func sampleHandler(w http.ResponseWriter, r *http.Request) {
	bucket := rand.IntN(100)
	if bucket > 49 {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))

	} else if bucket > 19 {
		time.Sleep(time.Second * 2)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("delayed response"))

	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/", sampleHandler)
	log.Fatal(http.ListenAndServe(":9000", nil))
}
