package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	DEFAULT_HTTP_PORT      int    = 80
	DEFAULT_HTTP_URI       string = "/health"
	DEFAULT_REDIS_ENDPOINT string = "127.0.0.1:6379"
	REFRESH_INTERVAL_MS    int    = 1000 // Milliseconds
	HTTP_RETURN_CODE_ERROR int    = 410  // Gone!
)

var (
	isEndpointHealthy = false
)

func main() {
	var (
		httpPort        int
		httpUri         string
		refreshInterval int
		redisEndpoint   string
		redisPassword   string
		err             error
		ctx             = context.Background()
		rdb             *redis.Client
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

	if envRedisEndpoint := os.Getenv("REDIS_ENDPOINT"); envRedisEndpoint != "" {
		redisEndpoint = envRedisEndpoint
	} else {
		redisEndpoint = DEFAULT_REDIS_ENDPOINT
	}
	fmt.Printf("[info] will check %s\n", redisEndpoint)

	if envRedisPassword := os.Getenv("REDIS_PASSWORD"); envRedisPassword != "" {
		redisPassword = envRedisPassword
		fmt.Printf("[info] will authenticate using given password\n")
	}

	// Init redis-client
	if len(redisPassword) > 0 {
		rdb = redis.NewClient(&redis.Options{
			Addr:     redisEndpoint,
			Password: redisPassword,
			DB:       0,
		})
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr: redisEndpoint,
			DB:   0,
		})
	}

	// Listen in the background
	go func(socket string) {
		fmt.Printf("[info] now running!\n")
		http.HandleFunc("/healthz", httpHandler)
		if err = http.ListenAndServe(socket, nil); err != nil {
			fmt.Printf("[err] %s\n", err)
			os.Exit(1)
		}
	}(":" + strconv.Itoa(httpPort))

	// Mainloop
	for {
		time.Sleep(time.Duration(refreshInterval) * time.Millisecond)
		pong, err := rdb.Ping(ctx).Result()
		if pong != "PONG" || err != nil {
			isEndpointHealthy = false
		} else {
			isEndpointHealthy = true
		}
	}
}

// Handler returns 200 by default
func httpHandler(w http.ResponseWriter, r *http.Request) {
	if !isEndpointHealthy {
		w.WriteHeader(HTTP_RETURN_CODE_ERROR)
		fmt.Fprintf(w, "ERROR\n")
	} else {
		fmt.Fprintf(w, "OK\n")
	}
}
