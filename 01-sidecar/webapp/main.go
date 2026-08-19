package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

var logPath = flag.String("logpath", "01-sidecar/shared/webapp.log", "path to the shared log file")

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func sampleHandler(w http.ResponseWriter, r *http.Request) {
	rec := &statusRecorder{
		ResponseWriter: w, statusCode: 200,
	}

	f, err := os.OpenFile(*logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o664)
	if err != nil {
		fmt.Println(err)
		http.Error(rec, "internal error", http.StatusInternalServerError)
		return
	}

	defer func() {
		_ = f.Close()
	}()

	defer func() {
		logline := fmt.Sprintf(" statuscode: %v, time: %v, method: %v, path: %v\n", rec.statusCode, time.Now(), r.Method, r.URL.Path)

		_, err := f.WriteString(logline)
		if err != nil {
			fmt.Println("Could not write to file: ", err)
		}
	}()

	_, err = rec.Write([]byte("logged\n"))
	if err != nil {
		fmt.Println("could no write to client: ", err)
	}
}

func main() {
	flag.Parse()
	http.HandleFunc("/", sampleHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
