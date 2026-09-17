# Scatter/Gather

## The idea

Sharding solved "how do I store more data than one machine can hold" — split the data across shards, and a router sends each request to the one shard that owns it. But that only works for requests that target a *specific* piece of data. What about a request that needs the *whole* picture — every key across the entire system, or a search across everything? No single shard has that; the answer is scattered across all of them.

Scatter/gather: instead of routing a request to one destination, send the same request to **every** shard at once, wait for all of them to answer, then combine their individual pieces into one final result.

```
                    ┌──> shard 1 --> partial result ──┐
client --> scatter ──> shard 2 --> partial result ──┼──> gather --> combined result
                    └──> shard 3 --> partial result ──┘
```

Classic real example: search engines. The index is sharded across many machines, nobody stores the whole thing on one box. A search query gets sent to every shard, each searches its own slice, and the results get merged before you see them.

## What's new here: doing things in parallel and waiting for all of them

Every pattern before this called out to one thing at a time. This is the first one where the router genuinely needs to do several things **simultaneously** and then wait until all of them finish before it can respond.

`go someFunc()` launches a goroutine and moves on immediately — it doesn't wait. That's exactly what we want for kicking off 3 parallel calls to 3 shards at once (3x faster than calling them one after another), but it means we also need a way to know when all 3 are actually done before we try to combine their results.

The tool for that here is a **buffered channel** — a pipe goroutines can send values into, sized to hold as many results as we're expecting (3, one per shard). Each shard-calling goroutine sends its result into the channel when it's done; the main handler receives from the channel exactly 3 times, and each receive blocks until a result is available. Buffering the channel to exactly the shard count matters: without it, a goroutine that finishes fast could get stuck waiting for the main handler to be ready to receive, even though the data itself was ready. With buffering, every goroutine can drop its result off and move on regardless of timing.

One classic Go trap worth knowing: when launching a goroutine per item in a loop, the shard URL is passed in as a parameter to the goroutine's closure rather than read directly from the loop variable. Goroutines run asynchronously, so by the time they actually execute, a loop variable read directly could have already moved on to a different iteration's value — passing it as an argument captures the correct value at launch time instead.

## What's in this folder

- `shard/` — same key-value store as the sharded-service exercise, plus one new endpoint: `GET /all`, which dumps every key-value pair that specific shard personally holds.
- `router/` — same hash-based routing as before for `/get` and `/set`, plus a new `GET /all` that scatters a request to all 3 shards concurrently, gathers their responses, and combines them into one answer. A shard that fails to respond doesn't take down the whole result — its failure is noted and gathering continues with whichever shards did answer.

## Running it on your own machine

You need Go 1.21+. Run everything from the repo root.

**Three terminals** — one shard each:
```
go run ./06-scatter-gather/shard -port=:9001
go run ./06-scatter-gather/shard -port=:9002
go run ./06-scatter-gather/shard -port=:9003
```

**One more terminal** — the router:
```
go run ./06-scatter-gather/router
```

**Set a handful of keys** (they'll land on different shards depending on their hash):
```
curl -X PUT "localhost:8080/set?key=apple&value=red"
curl -X PUT "localhost:8080/set?key=banana&value=yellow"
curl -X PUT "localhost:8080/set?key=cherry&value=dark-red"
curl -X PUT "localhost:8080/set?key=date&value=brown"
curl -X PUT "localhost:8080/set?key=elderberry&value=purple"
```

**Then scatter/gather everything:**
```
curl localhost:8080/all
```

You should get all 5 pairs back, combined into one response, even though no single shard holds all of them. To see the split for yourself, hit a shard's `/all` directly (`curl localhost:9001/all`) — you'll only see the subset of keys that particular shard happens to own.
