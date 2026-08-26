# Replicated Load-Balanced Service

## The idea

Everything built in the single node patterns (sidecar, ambassador, adapter) was about one logical unit, one main app, with one companion process helping it. This pattern is a different kind of problem: what happens when one instance of an app just isn't enough?

One instance can only handle so much traffic before it falls over, and if that one process crashes or its machine dies, every user is locked out, because there was never a second copy to fall back on.

The fix: run multiple identical copies of the same app ("replicas"), and put something in front of them that spreads incoming requests across all the copies a load balancer.

```
                    ┌──> webapp replica 1
client --> load balancer ──> webapp replica 2
                    └──> webapp replica 3
```

Two wins: more throughput (three replicas can roughly handle three times the traffic of one), and resilience (if one replica goes down, the others keep serving while it's ignored or replaced).

## Why this only works because webapp is stateless

Every replica needs to be able to handle any incoming request interchangeably a client shouldn't notice or care which replica actually answered. That only works if the app doesn't keep anything in its own memory between requests. `webapp` (from the sidecar exercise) qualifies: it logs and responds fresh every time, remembering nothing from one request to the next. That's what makes it safe to run three copies of side by side.

Contrast that with the adapter from the previous exercise, which keeps a running count in memory. If you replicated that as is, each replica would build up its own separate, disagreeing count, and there'd be no single correct answer to "how many requests total" anymore nothing would be lost, but there'd be three different sources of truth instead of one. That's a real problem in distributed systems: replicating something stateful doesn't lose your data, it splits your single source of truth into several that silently drift apart.

The load balancer itself, on the other hand, is fine being stateful here, because there's only one of it in this exercise no sibling copy it needs to agree with. The "don't make it stateful" concern is specifically about things you're replicating, not a blanket rule.

## What's in this folder

- `webapp/` — same app as the sidecar exercise, with one change: the port it listens on is now a flag instead of hardcoded, so multiple copies can run side by side without colliding.
- `loadbalancer/` — listens on one public facing port, and for every incoming request picks the next replica in a round-robin rotation, forwards the request there, and relays the response back.

## How the load balancer picks a replica

Round robin: cycle through the known replicas in order, wrapping back to the first once you reach the end. It keeps a `next` index in memory and advances it (`(next + 1) % number_of_replicas`) every time a request comes in.

Since the load balancer's own HTTP server handles requests concurrently (a goroutine per request, same as every server in this repo), multiple goroutines can be reading and advancing that index at the same time. Same shared-state problem as the adapter's counters, same fix: a mutex around the read-and-advance step.

## Running it on your own machine

You need Go 1.21+. Run everything from the repo root.

**Three terminals** — one webapp replica each, on different ports:
```
go run ./04-replicated-lb/webapp -port=:9001
go run ./04-replicated-lb/webapp -port=:9002
go run ./04-replicated-lb/webapp -port=:9003
```

**One more terminal** — the load balancer:
```
go run ./04-replicated-lb/loadbalancer
```

**Send some traffic through the load balancer** (not directly at the replicas):
```
for i in {1..9}; do curl -s localhost:8080/; done
```

All three replicas share the same log file, so `cat 04-replicated-lb/shared/webapp.log` will show all 9 requests. To actually confirm they were spread evenly across the three replicas rather than all landing on one, look at the `m=+...` value in each log line (Go's monotonic clock reading, roughly "seconds since that specific process started") — since each replica was started at a slightly different moment, each has its own distinct baseline, and you should see that baseline repeat in a strict 3-cycle pattern across the 9 log lines if round robin is working correctly.
