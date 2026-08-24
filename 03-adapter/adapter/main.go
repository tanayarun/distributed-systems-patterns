package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var logPath = flag.String("logpath", "03-adapter/shared/webapp.log", "path to the shared log file")

type counters struct {
	mu   sync.Mutex
	data map[string]int
}

func (c *counters) metricsHandler(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, _ = fmt.Fprintf(w, "# HELP http_requests_total Total number of HTTP requests\n")
	_, _ = fmt.Fprintf(w, "# TYPE http_requests_total counter\n")

	for key, count := range c.data {
		parts := strings.Split(key, "|")
		method := parts[0]
		path := parts[1]
		statusCode := parts[2]

		_, _ = fmt.Fprintf(w, "http_requests_total{method=%q,path=%q,status=%q} %d\n", method, path, statusCode, count)
	}
}

func (c *counters) increment(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key]++
}

func parseLine(line string) (statusCode string, method string, path string, err error) {
	parts := strings.Split(line, ",")
	if len(parts) != 4 {
		return "", "", "", fmt.Errorf("unexpected number of fields: %d", len(parts))
	}
	statusCode = parts[0]
	method = parts[2]
	path = parts[3]

	kv1 := strings.SplitN(statusCode, ": ", 2)
	if len(kv1) != 2 {
		return "", "", "", fmt.Errorf("unexpected number of fields: %d", len(kv1))
	}
	kv2 := strings.SplitN(method, ": ", 2)
	if len(kv2) != 2 {
		return "", "", "", fmt.Errorf("unexpected number of fields: %d", len(kv2))
	}
	kv3 := strings.SplitN(path, ": ", 2)
	if len(kv3) != 2 {
		return "", "", "", fmt.Errorf("unexpected number of fields: %d", len(kv3))
	}

	return kv1[1], kv2[1], kv3[1], nil
}

func main() {
	c := &counters{data: make(map[string]int)}

	flag.Parse()
	f, err := os.Open(*logPath)
	if err != nil {
		fmt.Println("could not open the file")
		return
	}

	defer func() {
		_ = f.Close()
	}()

	_, err = f.Seek(0, io.SeekEnd)
	if err != nil {
		fmt.Println("erorr seeking file")
		return
	}

	buf := make([]byte, 1024)
	leftover := ""

	go func() {
		for {
			n, err := f.Read(buf)

			if n > 0 {
				combined := leftover + string(buf[:n])
				lines := strings.Split(combined, "\n")

				completeLines := lines[:len(lines)-1]
				leftover = lines[len(lines)-1]

				for _, completeLine := range completeLines {
					statusCode, method, path, err := parseLine(completeLine)
					if err != nil {
						fmt.Println("skipping bad line:", err)
						continue // <-- skip just this line, not the whole program
					}
					key := method + "|" + path + "|" + statusCode
					c.increment(key)
				}
			}

			if err != nil {
				if err == io.EOF {
					time.Sleep(time.Second * 1)
				} else {
					fmt.Println("error reading buffer")
					return
				}
			}
		}
	}()

	http.HandleFunc("/metrics", c.metricsHandler)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
