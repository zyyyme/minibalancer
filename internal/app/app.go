package minibalancer

import (
	"context"
	"fmt"
	"log"
	"minibalancer/internal/services"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"time"
)

type ConfigHandler struct {
	config         services.Config
	servicesVitals map[string][]chan services.UsageData
}

func (handler *ConfigHandler) requestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := services.RedirectRequest(w, r, handler.config, handler.servicesVitals)

		if err != nil {
			http.Error(w, err.Error(), 500)
		}
	})
}

func (handler *ConfigHandler) setUpChannels() error {
	containers, err := services.GetContainersList()
	if err != nil {
		return err
	}

	grouppedContainerIDs := make(map[string][]services.ContainerInfo)

	for name, info := range containers {
		for _, service := range handler.config.Service {
			if strings.Contains(name, service.Name) {
				_, exists := grouppedContainerIDs[service.UrlPrefix]

				if exists {
					grouppedContainerIDs[service.UrlPrefix] = append(grouppedContainerIDs[service.UrlPrefix], info)
				} else {
					grouppedContainerIDs[service.UrlPrefix] = []services.ContainerInfo{info}
				}

				break
			}
		}
	}

	handler.servicesVitals = make(map[string][]chan services.UsageData)

	for name, containersInfo := range grouppedContainerIDs {
		handler.servicesVitals[name] = []chan services.UsageData{}

		for _, info := range containersInfo {
			newChannel := make(chan services.UsageData)

			go func(channel chan services.UsageData, id string, address string) {
				err := services.StreamStats(channel, id, address)
				if err != nil {
					fmt.Printf("Initialize streams error: %s\n", err)
				}
			}(newChannel, info.ID, info.Address)
			handler.servicesVitals[name] = append(handler.servicesVitals[name], newChannel)
		}
	}

	return nil
}

func ping(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(w, "pong")
}

func StartServer() {
	config, err := services.GetConfig("config.yaml")
	if err != nil {
		panic(err)
	}

	for _, i := range config.Service {
		fmt.Printf("Loaded service: %s, prefix: %s, replicas: %d\n", i.Name, i.UrlPrefix, len(i.ServerPool))
	}

	configHandler := &ConfigHandler{config: config}
	err = configHandler.setUpChannels()
	if err != nil {
		panic(err)
	}

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Server is starting...")

	router := http.NewServeMux()
	router.Handle("/", configHandler.requestHandler())
	router.HandleFunc("/ping", ping)

	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/debug/pprof/trace", pprof.Trace)

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
				logger.Println(r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}
