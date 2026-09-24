# Functions / Event-Driven Pattern

## The idea

A worker that continuously polls for work has to stay running all the time, even during the long stretches when there's nothing to do. If events only happen occasionally and unpredictably — a file gets uploaded once an hour, an order comes in twice a day — keeping a whole process alive 24/7 just to react to something rare feels wasteful.

Event-driven processing flips this: instead of a long-running worker actively waiting for work, you write a small, focused function that says "when event X happens, run me, with this event as input." Something else — an event source, a platform, a framework — is responsible for detecting that the event happened and invoking your function at that moment. The function runs, does its one job, and stops existing until the next event triggers it again. This is the idea behind serverless / FaaS (AWS Lambda, Google Cloud Functions, Knative) — you write a function, you never manage a server that's always running.

There are really two separate concerns bundled into this one pattern: the event source (what detects that something happened and decides to invoke code), and the function itself (the logic that runs once triggered, given the event's data).

## What's in this folder

One program this time — `eventsource/`, an HTTP server that receives events and dispatches them to the right function.

## The core mechanic: functions as values

The interesting part isn't the HTTP server, it's how dispatch works. Instead of a big if/else chain that has to know about every event type up front, event types map to actual functions:

```go
type EventHandler func(payload string) string

handlers := map[string]EventHandler{
    "user.created":  handleUserCreated,
    "file.uploaded": handleFileUploaded,
}
```

Go treats functions as values — they can be stored in a variable, passed around, and put in a map, the same as a string or an int. When an event arrives, the event source looks up `handlers[eventType]`, gets back the actual function, and calls it with the event's payload. Registering a new event type is a one-line addition to the map; the dispatch logic itself never needs to change or know what the function actually does.

This is genuinely the same mechanism real FaaS platforms use under the hood — the platform doesn't know your business logic ahead of time, it just knows to look up whatever's registered for an event type and invoke it when that type of event shows up.

## Running it on your own machine

You need Go 1.21+. Run from the repo root:

```
go run ./07-event-driven/eventsource
```

Send it some events:
```
curl -X POST localhost:8080/events -d '{"type":"user.created","payload":"someone@example.com"}'
curl -X POST localhost:8080/events -d '{"type":"file.uploaded","payload":"photo.png"}'
curl -X POST localhost:8080/events -d '{"type":"order.placed","payload":"order-123"}'
```

Each should dispatch to its own registered function and return quickly. Try an event type that was never registered:
```
curl -X POST localhost:8080/events -d '{"type":"nonexistent.event","payload":"x"}'
```

That one 404s — proof it's a real lookup-based registry deciding what runs, not a hardcoded chain silently falling through to nothing.
