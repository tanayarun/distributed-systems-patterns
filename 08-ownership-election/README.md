# Ownership Election

## The idea

Sometimes you run several replicas of a service so that if one dies, the others are still up. But some jobs must be done by exactly one instance at a time. A scheduled cleanup that deletes expired records, a write to something that can't handle two writers, being the one process that reacts to a certain event. If all three replicas run that job at once, you get duplicate or conflicting work. If none of them run it, the work never happens.

You still want the resilience, though. If the instance doing the job crashes, another replica should pick it up automatically, without a person stepping in.

The answer is leader election. Among a group of replicas, exactly one is the leader (or owner) at any moment. The leader does the exclusive work. The others sit idle but keep watching, and if the leader stops responding, one of them takes over.

## How it works: a lease

The simplest real-world way to do this, and the one Kubernetes itself uses for its controllers, is a lease. Think of it as a claim ticket with an expiry time.

1. There is one agreed-upon place that records who currently holds the lease.
2. A candidate can acquire the lease only if nobody holds it, or the current holder's lease has expired.
3. The leader has to keep renewing the lease before it expires. Each renewal says "I'm still alive."
4. If the leader crashes and stops renewing, the lease runs out. The next candidate to try acquiring it succeeds, and becomes the new leader.

There's no voting and no consensus algorithm here. It's just "first to claim an unclaimed or expired lease wins," checked over and over. Real distributed consensus (Raft, Paxos) is a much harder problem, and this pattern gets a lot done without it.

## What's in this folder

- `leaseserver/`: the shared authority. It tracks who holds the lease and when it expires. It has three endpoints: `/acquire?id=X`, `/renew?id=X`, and `/status`.
- `candidate/`: one replica competing for leadership. Every 2 seconds it checks in. If it's the leader it renews. If it isn't, it tries to acquire. It prints what it's doing so you can watch elections happen across terminals.

## Why a mutex matters here

Two candidates can hit `/acquire` at the same moment. If the server checked "is the lease free?" and then claimed it in two separate steps, both could see "free" and both could believe they won. So the whole check-then-claim sequence runs under a single mutex, which makes it atomic. It's the same idea as the router's round-robin index and the adapter's counters: whenever a read-then-write must behave as one step, lock around all of it.

`/renew` has a related safeguard. It only succeeds if the caller is actually the current holder, so a candidate that lost the lease can't extend it just by asking.

## Running it on your own machine

You need Go 1.21+. Run everything from the repo root. You'll want four terminals.

**Terminal 1**, the lease server:
```
go run ./08-ownership-election/leaseserver
```

**Terminals 2, 3, 4**, one candidate each (start them close together):
```
go run ./08-ownership-election/candidate -id=candidate-1
go run ./08-ownership-election/candidate -id=candidate-2
go run ./08-ownership-election/candidate -id=candidate-3
```

Exactly one should print `became leader`. The other two keep printing `not leader`. You can also check what the server thinks at any time:
```
curl localhost:8080/status
```

### Watch a failover

Ctrl+C the terminal of whichever candidate is leader. Then run `curl localhost:8080/status` right away. It will still show the dead candidate as leader, because the server can't know it died, only that its lease hasn't expired yet. Wait about 5 seconds and check again. Another candidate should now be the leader.

That delay is the main tradeoff of lease-based election. Failover takes about as long as the lease duration. A shorter lease means faster failover, but a healthy leader is more likely to lose the lease if a single renewal is slow.

## Limits worth knowing (not fixed here)

- **The lease server is a single point of failure.** If it goes down, nobody can acquire or renew. Real systems store the lease in something already replicated. Kubernetes leader election uses etcd, which is itself a consensus cluster.
- **A paused leader doesn't know it lost the lease.** If a leader freezes for longer than the lease duration (a long GC pause, a stalled VM), another candidate takes over. When the old leader wakes up it still believes it's the leader until its next renew fails, so for a moment two nodes think they own the job. The standard fix is a fencing token: a number that increases with every new leader, so the resource being protected can reject writes from the stale one. This exercise doesn't build that.
