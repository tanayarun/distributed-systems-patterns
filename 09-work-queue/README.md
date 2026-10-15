# Work Queue

## The idea

Some tasks are slow: resizing images, sending bulk emails, generating reports. If a client had to wait for an HTTP response until the whole task finished, it would be stuck for a long time, and the connection would stay tied up the entire while. Often the client doesn't even need to wait. It just wants to hand the task off and move on.

A work queue separates submitting work from doing work, with a queue in the middle:

```
producer --> [ task, task, task, task ] --> worker(s)
             (the queue)
```

- A **producer** adds a task to the queue and gets an ID back right away. It doesn't wait for the task to finish.
- One or more **workers** pull tasks off the queue and process them at their own pace.
- To get more throughput, add more workers. They all pull from the same queue, so the work spreads across them on its own.

That last point is a form of load balancing, but it works the opposite way from the replicated load balancer. There, something pushed requests at replicas. Here, workers ask for work when they're ready, so a slow worker just takes fewer tasks instead of getting buried.

## How this is different from every pattern before it

Everything up to now was request/response. A client sends a request and gets an answer back on the same connection. A work queue is asynchronous: submitting a task and finishing it are two separate events, possibly far apart in time. The producer doesn't get a result, because there isn't one yet.

So how does anyone find out how a task went? We used a task ID and a status endpoint. Submitting returns an ID immediately, and later anyone can ask `/status?id=X` to see whether that task is pending, running, or done, and what its result was. (The other common approach is for the worker to push the result somewhere else when it's done, like another queue or a webhook. We didn't build that.)

## What's in this folder

- `queue/`: the queue server. It keeps every task in a map by ID, plus a list of the IDs still waiting, in order. Endpoints:
  - `POST /submit?payload=X`: adds a task, returns its ID
  - `GET /pull`: hands the next pending task to a worker (or 204 if there's nothing to do)
  - `POST /complete?id=X&result=Y`: a worker reports a finished task
  - `GET /status?id=X`: check a task's status and result
- `worker/`: pulls a task, "processes" it (sleeps 2 seconds, then uppercases the payload, standing in for real work), reports the result, and pulls again. If the queue is empty it waits a second and tries again. Its name is a flag so you can run several.

## Why a task never goes to two workers

`/pull` takes the first ID off the pending list and marks that task as running, all under one mutex. So two workers pulling at the same moment can't both get the same task. Same idea as everywhere else in this repo: when a read and a write have to happen as one step, lock around both.

## A limit worth knowing about (not fixed here)

If a worker dies partway through a task, that task stays `running` forever, because nothing notices the worker is gone. The fix is the same lease idea from the ownership election exercise: a worker claims a task for a limited time, and if it doesn't report back before the time runs out, the task goes back to pending for someone else. Real queue systems (SQS's visibility timeout, for example) work this way. Also, all of this lives in memory, so restarting the queue server loses every task.

## Running it on your own machine

You need Go 1.21+. Run everything from the repo root.

**Terminal 1**, the queue server:
```
go run ./09-work-queue/queue
```

**Terminals 2 and 3**, two workers:
```
go run ./09-work-queue/worker -name=worker-1
go run ./09-work-queue/worker -name=worker-2
```

**Terminal 4**, submit a batch of tasks:
```
for i in {1..6}; do curl -s -X POST "localhost:8080/submit?payload=job-$i"; done
```

The submits return immediately with IDs, but each task takes 2 seconds to process. Watch the two worker terminals: the tasks split between them, and no task appears in both.

Check on a task using one of the IDs that came back:
```
curl "localhost:8080/status?id=1"
```

Right after submitting you'll see a mix of `pending`, `running`, and `done`. A few seconds later everything is `done`, with the uppercased payload as the result.
