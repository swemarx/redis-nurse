# redis-nurse

# What
An example of the Kubernetes "sidecar health"-pattern, healthchecking one or more Redis-instances in the same pod and exposing an endpoint presenting the results.

# How
You define your variables as:
```
$ export HTTP_PORT=9999
$ export HTTP_URI=/healthz/
$ export REFRESH_INTERVAL=5000
$ export REDIS_ENDPOINTS="1.2.3.4:6379 5.6.7.7:6379"
$ ./redis-nurse
```

The status of the first endpoint (1.2.3.4:6379) will be available at :::$HTTP_PORT/$HTTP_URI/0
```
$ curl localhost:9999/healthz/0
1.2.3.4:6379 is HEALTHY
```

The status of the second endpoint (5.6.7.8:6379) will be available at :::$HTTP_PORT/$HTTP_URI/1
```
$ curl localhost:9999/healthz/1
5.6.7.8:6379 is HEALTHY
```

If not specified, the following default values will be used:
```
HTTP_PORT=80
HTTP_URI=/health/
REDIS_ENDPOINTS=127.0.0.1:6379
REFRESH_INTERVAL=1000
REDIS_PASSWORD=""
```

The HTTP_URI variable needs to start and end with a backslash.
If specified, REDIS_PASSWORD will be used for all endpoints.
