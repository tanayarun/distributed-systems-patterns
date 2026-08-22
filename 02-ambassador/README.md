# Ambassador Pattern

## The idea

Suppose your app needs to call some other service over the network. In the real world that call can fail, be slow, need retries, need TLS, need service discovery, whatever. None of that is really about your app's actual logic. Your app just wants to say "get me this data." Someone still has to handle all that messiness though.

Two ways to handle it:

1. Write all that networking resilience logic directly into your app
2. Your app talks to `localhost`, as if the other service were right there, and a separate small program sitting next to it does the real network call, retries, timeouts, whatever's needed and hands back the result

Option 2 is the ambassador pattern. The ambassador is a local stand in for a remote service. Your app thinks it's talking to something simple and local. The ambassador does the hard part on its behalf.

## How this is different from the sidecar

In the sidecar exercise, the main app did its job completely on its own, and the sidecar just watched from the side. if the sidecar died, the main app wouldn't even notice. The ambassador is different: it sits directly **in the path** of the request. The main app is waiting on a response to do its own job, so if the ambassador goes down, the request fails. Sidecar is a bystander. Ambassador is a required hop.

What doesn't change: the main app still has no idea the ambassador exists. It just makes a normal HTTP call to a local address. No special API, no imported SDK, nothing ambassador-specific in its code. All the complexity is hidden behind that one address.

## What's in this folder

- `backend/` — a fake "remote service," intentionally flaky. ~50% of requests succeed fast, ~30% succeed but take 2 seconds, ~20% fail outright with a 500. This exists to give the ambassador something real to handle.
- `ambassador/` — sits between `webapp` and `backend`. Forwards each request to the backend, and if that attempt fails or times out, retries a few times (with a short delay between attempts) before giving up. If any attempt succeeds, it relays that response straight back.
- `webapp/` — the main app. Makes a plain HTTP call to the ambassador's address. Doesn't know retries or backoff exist behind it.

When I first wired this together, `webapp` was still using a 1 second timeout it originally had for calling the backend directly. Once the ambassador was inserted, that 1 second timeout was too short for the ambassador to actually finish its retries, so `webapp` was giving up and returning an error *before* the ambassador had a real chance to recover from a flaky backend call. The retries were happening, they just never got to matter, because the caller quit waiting first.

## Running it on your own machine

You need Go 1.21+. Run everything from the repo root.

**Terminal 1** — the flaky backend:
```
go run ./02-ambassador/backend
```

**Terminal 2** — the ambassador:
```
go run ./02-ambassador/ambassador
```

**Terminal 3** — the main app:
```
go run ./02-ambassador/webapp
```

**Terminal 4** — hammer it and watch:
```
for i in {1..20}; do curl -s -o /dev/null -w "%{http_code} %{time_total}s\n" localhost:8080/; done
```

You should see all 200s, no 500s, even though the backend underneath is still exactly as flaky as ever. Some requests will be near instant (succeeded on the first try), some will take a few hundred ms to a couple seconds (succeeded on a retry), that variance is the retry logic doing its job, invisibly, from `webapp`'s point of view.

If you want to see the pain this pattern is solving, point `webapp` at the backend directly (`http://localhost:9000/` instead of the ambassador's port) and run the same test you'll see a mix of fast 200s and 500s/timeouts.
