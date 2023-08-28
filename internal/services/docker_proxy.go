package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"strconv"
)

type UsageData struct {
	CpuUsage      uint64
	MemUsage      uint64
	ContainerName string
}

type ContainerInfo struct {
	Name    string
	ID      string
	Address string
}

var cli, _ = client.NewClientWithOpts(client.FromEnv)

func GetContainersList() (map[string]ContainerInfo, error) {
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})

	if err != nil {
		fmt.Printf("Container list error: %s\n", err.Error())
		return nil, err
	}

	containersInfo := make(map[string]ContainerInfo)
	for _, container := range containers {
		containersInfo[container.Names[0]] = ContainerInfo{
			Name:    container.Names[0],
			ID:      container.ID,
			Address: container.Ports[0].IP + ":" + strconv.Itoa(int(container.Ports[0].PublicPort)),
		}
	}

	return containersInfo, nil
}

// todo: do not pass address here, ugly solution
func StreamStats(statChannel chan UsageData, containerId string, containerAddress string) error {
	defer close(statChannel)

	dockerStream, err := cli.ContainerStats(context.Background(), containerId, true)
	defer dockerStream.Body.Close()

	decoder := json.NewDecoder(dockerStream.Body)

	if err != nil {
		return err
	}

	for {
		var unmarshalledData types.Stats

		err = decoder.Decode(&unmarshalledData)
		if err != nil {
			fmt.Println(containerAddress, err)
		}

		cpuStats := unmarshalledData.CPUStats.CPUUsage.TotalUsage
		memStats := unmarshalledData.MemoryStats.Usage

		statChannel <- UsageData{CpuUsage: cpuStats, MemUsage: memStats, ContainerName: containerAddress}
	}
}
