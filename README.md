# distributed-systems-patterns

implementations of the patterns from the book *Designing Distributed Systems* by Brendan Burns, written in Go.

Reading book is useless until you implement the concepts from scratch. Current implementation is in plain Go, and later I will re-implement the same patterns using real containers (Docker) and real Kubernetes pods. Each folder is one pattern, and each one is meant to be run and played with, not just read.

## Why I choose plain Go first?

Most of these patterns exist because of how containers and Kubernetes work shared volumes, shared network namespaces, sidecar containers in a pod, and so on.

1. tried to simulate the pattern with plain Go processes on my own machine (a shared folder stands in for a shared volume, localhost stands in for a shared network namespace, etc.)
2. once I understood the core idea, I will containerize the same thing with Docker
3. then deploy it for real on a Kubernetes cluster (using kind)

## Patterns

| Folder | Pattern | Status |
|---|---|---|
| [`01-sidecar`](./01-sidecar) | Sidecar | completed (plain Go) |
| [`02-ambassador`](./02-ambassador) | Ambassador | completed (plain Go) |
| [`03-adapter`](./03-adapter) | Adapter | completed (plain Go) |
| `04-replicated-lb` | Replicated Load-Balanced Service | planned |
| `05-sharded-service` | Sharded Service | planned |
| `06-scatter-gather` | Scatter/Gather | planned |
| `07-work-queue` | Work Queue | planned |
| `08-event-driven` | Event-Driven / Functions | planned |
| `09-leader-election` | Ownership Election | planned |

Each folder has its own README explaining the pattern and how to run it.

## Running any pattern

Unless a pattern's own README says otherwise, run everything from this repo's root, not from inside the pattern's subfolder, the code assumes that as the working directory.
