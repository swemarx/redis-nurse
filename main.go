package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
	"github.com/go-redis/redis/v8"
)

const (
	DEFAULT_HTTP_LISTEN_PORT int = 80
	DEFAULT_REDIS_ENDPOINT string = "127.0.0.1:6379"
	REFRESH_INTERVAL_MS int = 1000							// Milliseconds
	HTTP_RETURN_CODE_ERROR int = 410						// Gone!
)

var (
	ctx = context.Background()
	rdb *redis.Client
	isEndpointHealthy = false
)

func main() {
	var (
		listenPort int
		refreshInterval int
		redisEndpoint string
		redisPassword string
		err error
	)

	// Set variables from env
	if envListenPort := os.Getenv("LISTEN_PORT"); envListenPort != "" {
		listenPort, err = strconv.Atoi(envListenPort)
		if err != nil {
			fmt.Printf("[warn] LISTEN_PORT is garbage, using %d instead", DEFAULT_HTTP_LISTEN_PORT)
			listenPort = DEFAULT_HTTP_LISTEN_PORT
		}
	} else {
		listenPort = DEFAULT_HTTP_LISTEN_PORT
	}
	if envRefreshInterval := os.Getenv("REFRESH_INTERVAL"); envRefreshInterval != "" {
		refreshInterval, err = strconv.Atoi(envRefreshInterval)
		if err != nil {
			fmt.Printf("[warn] REFRESH_INTERVAL is garbage, using %d instead", REFRESH_INTERVAL_MS)
			refreshInterval = REFRESH_INTERVAL_MS
		}
	} else {
		refreshInterval = REFRESH_INTERVAL_MS
	}
	if envRedisEndpoint := os.Getenv("REDIS_ENDPOINT"); envRedisEndpoint != "" {
		redisEndpoint = envRedisEndpoint
	} else {
		redisEndpoint = DEFAULT_REDIS_ENDPOINT
	}
	if envRedisPassword := os.Getenv("REDIS_PASSWORD"); envRedisPassword != "" {
		redisPassword = envRedisPassword
	}

	// Init redis-client
	if len(redisPassword) > 0 {
		rdb = redis.NewClient(&redis.Options{
			Addr:		redisEndpoint,
			Password:	redisPassword,
			DB:			0,
		})
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr:		redisEndpoint,
			DB:			0,
		})
	}

	// Listen in the background
	go func(socket string) {
		fmt.Printf("Listening on %s\n", socket)
		http.HandleFunc("/healthz", httpHandler)
		http.ListenAndServe(socket, nil)
	}(":" + strconv.Itoa(listenPort))

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
