package services

import (
	"fmt"
	yamlparser "minibalancer/utils"
	"net/http"
	"strings"
)

type InstanceInfo struct {
	name    string
	ramUsed uint64
	cpuUsed uint64
}

type LoadInfo struct {
	serviceName string
	instances   []InstanceInfo
}

func RedirectRequest(w http.ResponseWriter, r *http.Request, config yamlparser.Config) {
	fmt.Printf("Incoming request: URI %s\n", r.RequestURI)
	mockLoad := LoadInfo{
		serviceName: "hard_service",
		instances: []InstanceInfo{
			{
				name:    "hard_service_instance_01",
				cpuUsed: 95,
				ramUsed: 4096,
			},
			{
				name:    "hard_service_instance_02",
				cpuUsed: 95,
				ramUsed: 4096,
			},
			{
				name:    "hard_service_instance_03",
				cpuUsed: 95,
				ramUsed: 4096,
			},
		},
	}

	for _, service := range config.Service {
		if strings.Contains(r.RequestURI, service.UrlPrefix) {
			fmt.Printf("Found needed service: %s\n", service.Name)
			minRam, minCpu := uint64(0), uint64(0) // todo: ineffective types
			var usedInstance string
			for _, instance := range mockLoad.instances {
				// todo: better criteria
				if minRam == 0 || minRam > instance.ramUsed || minCpu == 0 || minCpu > instance.cpuUsed {
					minRam = instance.ramUsed
					minCpu = instance.cpuUsed
					usedInstance = instance.name
				}
			}
			fmt.Printf("Selected instance %s as least loaded", usedInstance)

			// todo: send request and get response
			fmt.Fprintf(w, "Instance: %s", usedInstance)
			return
		}
	}
	http.Error(w, "not found", 404)
}
