package services

import (
	yamlparser "minibalancer/utils"
	"net/http"
	"strings"
)

type LoadInfo struct {
	serviceName string
	instances   []UsageData
}

func RedirectRequest(w http.ResponseWriter, r *http.Request, config yamlparser.Config, usages []UsageData) {

	for _, service := range config.Service {
		if strings.Contains(r.RequestURI, service.UrlPrefix) {
			minRam, minCpu := uint64(0), uint64(0) // todo: ineffective types
			var usedInstance string
			for _, instance := range usages {
				// todo: better criteria
				if minRam == 0 || minRam > instance.MemUsage || minCpu == 0 || minCpu > instance.CpuUsage {
					minRam = instance.MemUsage
					minCpu = instance.CpuUsage
					usedInstance = instance.ContainerName
				}
			}

			// todo: send request and get response
			SendRequest(w, r, service.UrlPrefix, usedInstance)
			return
		}
	}
	http.Error(w, "not found", 404)
}
