# httppub

While `nginx` has the ability to `proxy_pass` an incoming request to any ONE of the configured `workers`. `httppub` has the ability to duplicate an incoming request to ALL of the configured `targets`.

### Why

You don't want to disrupt an existing service for existing clients, but would like those traffic to go to other servers too (e.g. to test alternative implementations or deployments).

`httppub` can let your existing http client & server to remain undisrupted, while duplicating incoming requests and send to other http servers.

### How

Given an incoming request `http://httppub.example.com/hello/world?key=value`, a target url `https://server/base` will receive a HTTPS request at `https://server/base/hello/world?key=value` with the same HTTP method, headers and request body.

#### Primary target

The first url in `-targets` option is the designated "primary" target.
- When the primary target responds, `httppub` will respond using its http status code, headers and response body. Responses from the other targets are discarded; they can even be down or unreachable.
- `httppub` will return a response once the `primary` target responds, without waiting for any other targets.


#### Target options

If any target is configured with a non-empty fragment, e.g. `http://server/base#fixed`, then the outgoing request will be `https://server/base?key=value` (note: `/hello/world` is dropped)

If any target is configured with query key-values, e.g. `https://server/base?Alpha=A&Beta=B`, then the outgoing request will carry request headers:

```
Alpha: A
Beta: B
```

### Usage

```
$ go run main.go -h
Usage of main:
  -addr string
    	address to listen at (default ":3000")
  -targets string
    	comma separated target urls; first url is primary (default "http://localhost:5000,http://127.0.0.1:5001/pre/fix")
  -timeout duration
    	maximum target request duration; maximum delay for shutdown of main app (default 30s)
```

The `-targets` option is a comma separated list of URLs to broadcast the incoming request to. If there are 4 urls in `-targets`, then for every 1x incoming request, 4x outgoing requests will be made concurrently.

### Notes

Thanks https://github.com/chrislusf/teeproxy for being there when I needed it. But I need more than A and B (so many serverless solutions out there!).

### License

MIT
