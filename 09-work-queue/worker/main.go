package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type task struct {
	ID      int
	Payload string
	Status  string
	Result  string
}

var (
	name      = flag.String("name", "worker-1", "this worker's name")
	queueAddr = flag.String("queue", "http://localhost:8080", "queue server address")
)

func main() {
	flag.Parse()
	client := &http.Client{Timeout: 5 * time.Second}

	for {
		resp, err := client.Get(*queueAddr + "/pull")
		if err != nil {
			fmt.Printf("[%s] could not reach queue: %v\n", *name, err)
			time.Sleep(time.Second)
			continue
		}

		if resp.StatusCode == http.StatusNoContent {
			_ = resp.Body.Close()
			time.Sleep(time.Second)
			continue
		}

		var t task
		err = json.NewDecoder(resp.Body).Decode(&t)
		_ = resp.Body.Close()
		if err != nil {
			fmt.Printf("[%s] could not decode task: %v\n", *name, err)
			time.Sleep(time.Second)
			continue
		}

		fmt.Printf("[%s] picked up task %d: %s\n", *name, t.ID, t.Payload)
		time.Sleep(2 * time.Second)
		result := strings.ToUpper(t.Payload)

		params := url.Values{}
		params.Set("id", strconv.Itoa(t.ID))
		params.Set("result", result)

		cresp, err := client.Post(*queueAddr+"/complete?"+params.Encode(), "", nil)
		if err != nil {
			fmt.Printf("[%s] could not report task %d: %v\n", *name, t.ID, err)
			continue
		}
		_ = cresp.Body.Close()
		fmt.Printf("[%s] finished task %d\n", *name, t.ID)
	}
}
