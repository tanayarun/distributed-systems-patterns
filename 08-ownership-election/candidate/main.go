package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	id          = flag.String("id", "candidate-1", "this candidate's unique id")
	leaseServer = flag.String("leaseserver", "http://localhost:8080", "lease server address")
)

func post(client *http.Client, url string) (int, string) {
	resp, err := client.Post(url, "", nil)
	if err != nil {
		return 0, err.Error()
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func main() {
	flag.Parse()
	client := &http.Client{Timeout: 2 * time.Second}

	isLeader := false

	for {
		if isLeader {
			status, body := post(client, *leaseServer+"/renew?id="+*id)
			if status == http.StatusOK {
				fmt.Printf("[%s] still leader, renewed: %s", *id, body)
			} else {
				fmt.Printf("[%s] LOST leadership (renew failed): %s\n", *id, body)
				isLeader = false
			}
		} else {
			status, body := post(client, *leaseServer+"/acquire?id="+*id)
			if status == http.StatusOK {
				fmt.Printf("[%s] became leader: %s", *id, body)
				isLeader = true
			} else {
				fmt.Printf("[%s] not leader, someone else holds it\n", *id)
			}
		}

		time.Sleep(2 * time.Second)
	}
}
