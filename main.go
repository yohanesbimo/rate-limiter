package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"rate-limiter/controller"
	"rate-limiter/middleware"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})

	// rateLimitMiddleware := middleware.NewRateLimiter(1, 5)
	redisRateLimiter := middleware.NewRedisRateLimiter(redisClient)

	router := gin.Default()
	// router.GET("/", rateLimitMiddleware.RateLimitChecker(), controller.TestRateLimiter)
	router.GET("/", redisRateLimiter.RateLimitChecker(), controller.RateLimiterController)

	srv := &http.Server{
		Addr:    ":3000",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	<-ctx.Done()

	stop()
	log.Println("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("forced to shutdown: ", err)
	}

	log.Println("server shutdown")
}
