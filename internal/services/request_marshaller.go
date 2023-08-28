package services

import (
	"errors"
	"fmt"
	"github.com/motemen/go-loghttp"
	"io"
	"net/http"
	"strings"
	"sync"
)

type LoadInfo struct {
	serviceName string
	instances   []UsageData
}

func RedirectRequest(w http.ResponseWriter, r *http.Request, config Config, usages map[string][]chan UsageData) error {
	for _, service := range config.Service {
		if strings.Contains(r.RequestURI, service.UrlPrefix) {
			minRam, minCpu := uint64(0), uint64(0) // todo: ineffective types
			var usedInstance string
			for _, instanceChannel := range usages[r.RequestURI] {
				// todo: better criteria
				instance := <-instanceChannel
				if minRam == 0 || minRam > instance.MemUsage || minCpu == 0 || minCpu > instance.CpuUsage {
					minRam = instance.MemUsage
					minCpu = instance.CpuUsage
					usedInstance = instance.ContainerName
				}
			}

			return SendRequest(w, r, service.UrlPrefix, usedInstance)
		}
	}

	return errors.New("not found")
}

func SendRequest(w http.ResponseWriter, r *http.Request, prefix string, containerId string) error {
	var wg sync.WaitGroup

	var res *http.Response

	wg.Add(1)
	go func() {
		defer wg.Done()

		forwardClient := &http.Client{
			Transport: &loghttp.Transport{},
		}

		requestUrl := "http://" + containerId + strings.TrimPrefix(r.RequestURI, prefix)
		request, err := http.NewRequest(r.Method, requestUrl, r.Body)
		if err != nil {
			panic(err)
		}

		request.Header = r.Header.Clone()
		res, err = forwardClient.Do(request)
		if err != nil {
			fmt.Println(err)
		}
	}()

	wg.Wait()
	for name, values := range res.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	return pipeReq(w, res)
}

func pipeReq(rw http.ResponseWriter, resp *http.Response) error {
	rw.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	rw.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	_, err := io.Copy(rw, resp.Body)
	if err != nil {
		return err
	}
	err = resp.Body.Close()
	if err != nil {
		return err
	}

	return nil

}
