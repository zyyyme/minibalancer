package main

import (
	"context"
	"fmt"
	"log"
	"minibalancer/services"
	"minibalancer/utils"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type key int

const (
	requestIDKey key = 0
)

type ConfigHandler struct {
	config utils.Config
}

func (handler *ConfigHandler) requestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usages := services.GetContainersInfo()

		services.RedirectRequest(w, r, handler.config, usages)
	})
}

func main() {
	config, err := utils.GetConfig("config.yaml")
	if err != nil {
		panic(err)
	}

	for _, i := range config.Service {
		fmt.Printf("Loaded service: %s, prefix: %s, replicas: %d\n", i.Name, i.UrlPrefix, len(i.ServerPool))
	}

	configHandler := &ConfigHandler{config: config}

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Server is starting...")

	router := http.NewServeMux()
	router.Handle("/", configHandler.requestHandler())

	server := http.Server{
		Addr:         "0.0.0.0:8080",
		Handler:      logging(logger)(router),
		ErrorLog:     logger,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		logger.Println("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	logger.Println("Server is ready to handle requests at 8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", "8080", err)
	}

	<-done
	logger.Println("Server stopped")
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Println("incoming", r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent(), ok)
			}()
			next.ServeHTTP(w, r)
		})
	}
}
