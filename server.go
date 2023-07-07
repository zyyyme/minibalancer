package main

import (
	"fmt"
	"log"
	"minibalancer/services"
	"minibalancer/utils"
	"net/http"
)

type ConfigHandler struct {
	config yamlparser.Config
}

func (handler *ConfigHandler) requestHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("TODO: Process request\n")
	services.RedirectRequest(w, r, handler.config)
}

func main() {

	config, err := yamlparser.GetConfig("config.yaml")
	if err != nil {
		panic(err)
	}

	for _, i := range config.Service {
		fmt.Printf("Loaded service: %s, prefix: %s, replicas: %d\n", i.Name, i.UrlPrefix, len(i.ServerPool))
	}

	configHandler := &ConfigHandler{config: config}

	fmt.Printf("TODO: load and validate config\n")

	http.HandleFunc("/", configHandler.requestHandler)

	fmt.Printf("Starting server at port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
