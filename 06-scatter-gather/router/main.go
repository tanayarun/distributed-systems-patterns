package main

import (
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"strings"
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
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "could not read body", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(body)
}

func (rt *router) callShard(shardURL string) (string, error) {
	resp, err := rt.client.Get(shardURL + "/all")
	if err != nil {
		return "", fmt.Errorf("calling %s: %w", shardURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading body from %s: %w", shardURL, err)
	}
	return string(body), nil
}

func (rt *router) scatterGatherHandler(w http.ResponseWriter, r *http.Request) {
	type result struct {
		shardURL string
		body     string
		err      error
	}

	results := make(chan result, len(rt.shards))

	for _, shardURL := range rt.shards {
		go func(shardURL string) {
			body, err := rt.callShard(shardURL)
			results <- result{shardURL: shardURL, body: body, err: err}
		}(shardURL)
	}

	var combined strings.Builder
	for i := 0; i < len(rt.shards); i++ {
		res := <-results
		if res.err != nil {
			fmt.Fprintf(&combined, "# shard %s failed: %v\n", res.shardURL, res.err)
			continue
		}
		combined.WriteString(res.body)
	}

	_, _ = w.Write([]byte(combined.String()))
}

func main() {
	rt := &router{
		shards: []string{"http://localhost:9001", "http://localhost:9002", "http://localhost:9003"},
		client: &http.Client{Timeout: 2 * time.Second},
	}
	http.HandleFunc("/get", rt.routerHandler)
	http.HandleFunc("/set", rt.routerHandler)
	http.HandleFunc("/all", rt.scatterGatherHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
