# Sidecar Pattern

## The idea

Suppose you have a web app, and someone asks you to also log every request and ship those logs to some central place. You've got two options:

1. Write that logging/shipping logic directly inside your web app
2. Write a second, small, separate program whose only job is logging/shipping, and run it alongside your web app

Many guys would choose option 1 first because it feels simple. But there are some issues: your app's code is now coupled up with something that has nothing to do with its actual job, a bug in the logging code can take down your whole app, and if you want to change how logs are shipped you have to touch and redeploy the main app.

Option 2 is the sidecar pattern. Two small programs, each doing one job, running next to each other, connected only through something they both have access to, in this case, a shared file.

The important rule: **the main app doesn't know or care that the sidecar exists.** It just does its normal job (writes a log file, like it always would). The sidecar happens to be watching that file and doing something useful with it. You could swap the sidecar for a completely different one, or delete it entirely, and the main app wouldn't need a single line changed.

In Kubernetes terms, a sidecar is a second container in the same pod as your main container. They share fate (if the pod dies, both die) and they share resources (like a volume), but not code. we're building here on a laptop with two Go processes and a shared folder is the exact same idea, without Kubernetes.

## What's in this folder

- `webapp/` — a tiny HTTP server. On every request it logs the timestamp, method, path, and response status code to a file. It has no idea anything is reading that file.
- `logshipper/` — the sidecar. It watches that same file and prints new lines as they show up, the same way `tail -f` works. It has no idea what generated those lines.
- `shared/` — alternate for a Kubernetes shared volume. Just a plain folder both programs read/write to.

## How the sidecar actually watches the file

It doesn't poll the whole file over and over. It opens the file, seeks straight to the end (so it ignores anything that was already there), and then loops: try to read new bytes, if there's nothing new yet (EOF) sleep for a bit and try again, if there's something new, process it. That's the entire trick behind `tail -f`.

## Running it on your own machine

You need Go 1.21+ installed. Run everything from the **repo root** (not from inside this folder), that's what the default paths assume.

**Terminal 1** — start the web app:
```
go run ./01-sidecar/webapp
```

**Terminal 2** — start the sidecar:
```
go run ./01-sidecar/logshipper
```

**Terminal 3** — send it some requests:
```
curl localhost:8080/
curl localhost:8080/foo
curl -X POST localhost:8080/bar
```

Watch terminal 2 — you'll see each request show up there shortly after you curl it, live.

### Changing where the shared log file lives

Both programs accept a `-logpath` flag if you don't want to use the default location:

```
go run ./01-sidecar/webapp -logpath=/tmp/mylog.log
go run ./01-sidecar/logshipper -logpath=/tmp/mylog.log
```

Just make sure you pass the **same path** to both, there's nothing in the code that enforces this, it's on you (or in a real deployment, on your config) to keep them in sync.
