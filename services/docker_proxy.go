package services

import (
	"context"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/motemen/go-loghttp"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type UsageData struct {
	CpuUsage      uint64
	MemUsage      uint64
	ContainerName string
}

func GetContainersInfo() []UsageData {
	cli, err := client.NewClientWithOpts(client.FromEnv)

	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{Filters: filters.NewArgs(filters.KeyValuePair{Key: "network", Value: "hyperhash_default"})})

	if err != nil {
		panic(err)
	}

	var usages []UsageData
	for _, container := range containers {
		containerStats, err := cli.ContainerStats(context.Background(), container.ID, false)

		if err != nil {
			panic(err)
		}

		var unmarshalledData types.Stats
		err = json.NewDecoder(containerStats.Body).Decode(&unmarshalledData)
		if err != nil {
			return nil
		}
		cpuStats := unmarshalledData.CPUStats.CPUUsage.TotalUsage
		memStats := unmarshalledData.MemoryStats.Usage

		usages = append(usages, UsageData{cpuStats, memStats, container.Ports[0].IP + ":" + strconv.Itoa(int(container.Ports[0].PublicPort))})
	}

	return usages
}

func SendRequest(w http.ResponseWriter, r *http.Request, prefix string, containerId string) {
	forwardClient := &http.Client{
		Transport: &loghttp.Transport{},
	}

	requestUrl := "http://" + containerId + strings.TrimPrefix(r.RequestURI, prefix)
	request, err := http.NewRequest(r.Method, requestUrl, r.Body)
	if err != nil {
		panic(err)
	}

	request.Header = r.Header.Clone()
	res, err := forwardClient.Do(request)
	if err != nil {
		panic(err)
	}

	for name, values := range res.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	pipeReq(w, res)
}

func pipeReq(rw http.ResponseWriter, resp *http.Response) {
	rw.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	rw.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	_, err := io.Copy(rw, resp.Body)
	if err != nil {
		return
	}
	err = resp.Body.Close()
	if err != nil {
		return
	}

}
