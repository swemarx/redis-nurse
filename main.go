package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DEFAULT_HTTP_PORT      int    = 80
	DEFAULT_HTTP_URI       string = "/health/"
	DEFAULT_REDIS_ENDPOINT string = "127.0.0.1:6379"
	REFRESH_INTERVAL_MS    int    = 1000 // Milliseconds
	HTTP_RETURN_CODE_BAD   int    = 400  // Bad request
	HTTP_RETURN_CODE_ERROR int    = 410  // Gone!
)

type RedisInstance struct {
	endpoint  string
	client    *redis.Client
	isHealthy bool
}

var (
	httpUri   string
	instances []*RedisInstance
)

func main() {
	var (
		httpPort        int
		refreshInterval int
		redisPassword   string
		err             error
		ctx             = context.Background()
	)

	// Set variables from env
	if envHttpPort := os.Getenv("HTTP_PORT"); envHttpPort != "" {
		httpPort, err = strconv.Atoi(envHttpPort)
		if err != nil || httpPort < 0 {
			fmt.Printf("[warn] HTTP_PORT is garbage, using %d instead\n", DEFAULT_HTTP_PORT)
			httpPort = DEFAULT_HTTP_PORT
		}
	} else {
		httpPort = DEFAULT_HTTP_PORT
	}
	fmt.Printf("[info] will listen to port %d\n", httpPort)

	if envHttpUri := os.Getenv("HTTP_URI"); envHttpUri != "" {
		httpUri = envHttpUri
	} else {
		httpUri = DEFAULT_HTTP_URI
	}
	fmt.Printf("[info] will use uri %s\n", httpUri)

	if envRefreshInterval := os.Getenv("REFRESH_INTERVAL"); envRefreshInterval != "" {
		refreshInterval, err = strconv.Atoi(envRefreshInterval)
		if err != nil || refreshInterval < 0 {
			fmt.Printf("[warn] REFRESH_INTERVAL is garbage, using %d instead\n", REFRESH_INTERVAL_MS)
			refreshInterval = REFRESH_INTERVAL_MS
		}
		if refreshInterval < 500 {
			fmt.Printf("[warn] REFRESH_INTERVAL is too low, using %d instead\n", REFRESH_INTERVAL_MS)
		}
	} else {
		refreshInterval = REFRESH_INTERVAL_MS
	}
	fmt.Printf("[info] will perform healthchecks every %d ms\n", refreshInterval)

	if envRedisEndpoints := os.Getenv("REDIS_ENDPOINTS"); envRedisEndpoints != "" {
		fmt.Println("REDIS_ENDPOINTS:")
		for _, e := range strings.Split(envRedisEndpoints, " ") {
			fmt.Printf("- %s\n", e)
			i := &RedisInstance{endpoint: e}
			instances = append(instances, i)
		}
	} else {
		instances = append(instances, &RedisInstance{endpoint: DEFAULT_REDIS_ENDPOINT})
	}

	if envRedisPassword := os.Getenv("REDIS_PASSWORD"); envRedisPassword != "" {
		redisPassword = envRedisPassword
		fmt.Println("[info] will authenticate using given password")
	} else {
		redisPassword = ""
		fmt.Println("[info] will not authenticate")
	}

	// Init redis-clients
	for _, i := range instances {
		i.client = redis.NewClient(&redis.Options{
			Addr:     i.endpoint,
			Password: redisPassword,
			DB:       0,
		})
	}

	// Listen in the background
	go func(socket string) {
		fmt.Printf("[info] now running!\n")
		http.HandleFunc(httpUri, httpHandler)
		if err = http.ListenAndServe(socket, nil); err != nil {
			fmt.Printf("[err] %s\n", err)
			os.Exit(1)
		}
	}(":" + strconv.Itoa(httpPort))

	//
	// Mainloop
	//
	for {
		for _, i := range instances {
			pong, err := i.client.Ping(ctx).Result()
			if pong != "PONG" || err != nil {
				fmt.Printf("[warn] %s\n", err)
				i.isHealthy = false
			} else {
				i.isHealthy = true
			}
		}
		time.Sleep(time.Duration(refreshInterval) * time.Millisecond)
	}
}

// Handler returns 200 by default. Kubernetes liveness/readiness checks only care about the http returncode, not the text.
func httpHandler(w http.ResponseWriter, r *http.Request) {
	idxstr := strings.TrimPrefix(r.URL.Path, httpUri)
	idx, err := strconv.Atoi(idxstr)
	if err != nil {
		fmt.Println("[error] Could not convert requested index into integer")
		w.WriteHeader(HTTP_RETURN_CODE_BAD)
		fmt.Fprintf(w, "BAD REQUEST\n")
		return
	}
	l := len(instances)
	if idx < 0 || idx > l-1 {
		fmt.Println("[error] Invalid index requested")
		w.WriteHeader(HTTP_RETURN_CODE_BAD)
		fmt.Fprintf(w, "BAD REQUEST\n")
		return
	}
	instance := instances[idx]
	if !instance.isHealthy {
		w.WriteHeader(HTTP_RETURN_CODE_ERROR)
		fmt.Fprintf(w, "%s is UNHEALTHY\n", instance.endpoint)
	} else {
		fmt.Fprintf(w, "%s is HEALTHY\n", instance.endpoint)
	}
}
