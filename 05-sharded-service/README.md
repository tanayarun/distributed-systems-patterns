# Sharded Service

## The idea

Replicated load balancing works when every copy of your app can answer any request equally well that only holds if the app is stateless. But some apps genuinely need to hold data, and that data can get too big or too busy for one machine to handle. You can't just replicate a big dataset three times and call it a day; keeping full copies in sync is expensive and messy.

The alternative: split the data into pieces ("shards"), and give each replica ownership of just one piece. No single shard holds the whole dataset — but all the shards together do.

```
                    ┌──> shard 1 (owns some keys)
client --> router ──> shard 2 (owns other keys)
                    └──> shard 3 (owns the rest)
```

The new problem this creates: unlike round robin, where any replica could serve any request, here the router has to figure out exactly which shard owns the specific piece of data being asked for, and send the request there and only there. Get the routing wrong and the data simply isn't on the shard you asked.

## How shards get decided: hashing

We used hash-based sharding: run the key through a hash function (FNV, from Go's standard library), which turns any string into a big number that looks essentially random but is always the *same* number for the same input. Then `hash(key) % number_of_shards` picks the owning shard.

Determinism is the whole point — the same key has to route to the same shard every single time, on both the write and every future read, or the data becomes unreachable. Good spread matters too — a hash function scrambles keys well enough that they land roughly evenly across shards, unlike naive alphabetical/range-based sharding, which can pile up unevenly if your real key distribution isn't uniform.

## What's in this folder

- `shard/` — a tiny in-memory key-value store. `PUT /set?key=X&value=Y` stores a value, `GET /get?key=X` reads it back. Mutex-protected map, same concurrency concern as everywhere else in this repo. Port is a flag so multiple instances can run side by side.
- `router/` — computes `hash(key) % number_of_shards` for every incoming request and forwards it to the one shard that owns it, preserving the original HTTP method (GET or PUT) and the full query string, then relays the response back.

## A wrinkle worth knowing: forwarding the original method

The replicated load balancer only ever forwarded GET requests, so it could get away with a plain `client.Get(url)`. The router here has to forward whatever method the original request actually used — GET for reads, PUT for writes — or every write would silently turn into a read and break. The fix is `http.NewRequest(r.Method, url, nil)` followed by `client.Do(req)`, which is the general-purpose way to send a request of any method; `client.Get` was just a shortcut for the GET case all along.

## A real limitation worth knowing about (not fixed here)

This uses plain `hash(key) % N`. If you ever change the number of shards — add a fourth, remove one — almost every key's `hash % N` result changes, meaning nearly all your data effectively "moves" to a different shard overnight, even though nothing about the data itself changed. Real systems solve this with consistent hashing, which limits how much data has to move when the shard count changes. Worth knowing this exists; not something this exercise needed to solve.

## Running it on your own machine

You need Go 1.21+. Run everything from the repo root.

**Three terminals** — one shard instance each:
```
go run ./05-sharded-service/shard -port=:9001
go run ./05-sharded-service/shard -port=:9002
go run ./05-sharded-service/shard -port=:9003
```

**One more terminal** — the router:
```
go run ./05-sharded-service/router
```

**Set some keys and read them back, through the router:**
```
curl -X PUT "localhost:8080/set?key=apple&value=red"
curl -X PUT "localhost:8080/set?key=banana&value=yellow"
curl "localhost:8080/get?key=apple"
curl "localhost:8080/get?key=banana"
```

Every key should come back correctly no matter how many times you ask, since the router always sends the same key to the same shard.

**To see the actual partitioning happen**, bypass the router and hit a shard directly:
```
curl "localhost:9001/get?key=apple"
```

If `9001` isn't the shard `apple` actually hashes to, you'll get a 404 — not because anything's broken, but because that shard genuinely never received that key. Only the shard the router's hash function picked will have it.
