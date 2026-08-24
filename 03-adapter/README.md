# Adapter Pattern

## The idea

Suppose you're running a bunch of different apps, written by different teams, maybe software you don't even control the code of. Your monitoring system expects metrics in one specific format. But every app you run outputs its stats in its own different, often messy format. You've got three options:

1. Change every app's code to output the format your monitoring system wants
2. Teach your monitoring system to understand every app's different format
3. For each app, run a small separate program next to it whose only job is: read that app's native output, translate it into the standard format your monitoring system expects, and expose that. The app doesn't change. The monitoring system doesn't change. The translator in the middle absorbs the mismatch.

Option 3 is the adapter pattern.

## How this is different from sidecar and ambassador

Adapter is actually closest to the sidecar, not the ambassador. The main app produces something on its own, with zero awareness of who's reading it. Same as sidecar.

The real difference from sidecar: a sidecar just relays or ships what it reads, without changing its shape. An adapter's entire reason for existing is to **transform** what it reads into a specific standard shape that something else requires. The sidecar's `logshipper` could print however it wanted, nothing downstream demanded a particular format from it. The adapter here has a real format to satisfy anything scraping `/metrics` expects genuine Prometheus exposition format.

Ambassador, for contrast, sits directly in the path of an outgoing call the main app makes and depends on for a response. Adapter doesn't sit in anyone's critical path, if it goes down, the main app keeps working fine, same as with a sidecar.

## What's in this folder

- `webapp/` — the exact same main app from `01-sidecar`, unchanged, still writing logs in its own arbitrary custom format, still with zero idea anything reads that file.
- `adapter/` — tails that same log file (same tailing technique as the sidecar's `logshipper`), parses each line, keeps a running count per (method, path, status) combination, and exposes those counts over HTTP at `/metrics` in real Prometheus exposition format.

## The concurrency piece worth understanding

This is the first pattern where two different things need the same data at the same time: the file tailing loop is constantly updating the counts in the background, while an HTTP request to `/metrics` might read those same counts at any moment. Two different goroutines touching the same map is a race condition waiting to happen — Go doesn't protect you from this automatically.

## Running it on your own machine

You need Go 1.21+. Run everything from the repo root.

**Terminal 1** — the main app:
```
go run ./03-adapter/webapp
```

**Terminal 2** — the adapter:
```
go run ./03-adapter/adapter
```

**Terminal 3** — generate some traffic:
```
curl localhost:8080/
curl localhost:8080/foo
curl -X POST localhost:8080/bar
```

**Terminal 4** — check the translated output:
```
curl localhost:8081/metrics
```

You should see real Prometheus-format counters reflecting exactly what you curled, something like:
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/",status="200"} 1
http_requests_total{method="POST",path="/bar",status="200"} 1
```

Nothing about `webapp`'s code changed to make this possible, and any real Prometheus server could scrape this endpoint as is.
