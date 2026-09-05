package main

import (
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"time"
)

type router struct {
	shards []string
	client *http.Client
}

func hashKey(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

func (rt *router) routerHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")

	shardIndex := hashKey(key) % uint32(len(rt.shards))

	url := rt.shards[shardIndex] + r.URL.Path + "?" + r.URL.RawQuery

	req, err := http.NewRequest(r.Method, url, nil)
	if err != nil {
		http.Error(w, "could not make request", http.StatusInternalServerError)
		return
	}
	resp, err := rt.client.Do(req)
	if err != nil {
		http.Error(w, "request failed", http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "could not read body", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(body)
}

func main() {
	rt := &router{
		shards: []string{"http://localhost:9001", "http://localhost:9002", "http://localhost:9003"},
		client: &http.Client{Timeout: 2 * time.Second},
	}
	http.HandleFunc("/get", rt.routerHandler)
	http.HandleFunc("/set", rt.routerHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
