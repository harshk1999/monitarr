package docker_helper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
)

type Container struct {
	ID              string          `json:"Id"`
	Names           []string        `json:"Names"`
	Image           string          `json:"Image"`
	ImageID         string          `json:"ImageID"`
	Command         string          `json:"Command"`
	Created         int64           `json:"Created"`
	State           string          `json:"State"`
	Status          string          `json:"Status"`
	Ports           []Port          `json:"Ports"`
	SizeRw          int64           `json:"SizeRw"`
	SizeRootFS      int64           `json:"SizeRootFs"`
	HostConfig      HostConfig      `json:"HostConfig"`
	NetworkSettings NetworkSettings `json:"NetworkSettings"`
	Mounts          []Mount         `json:"Mounts"`
}

type HostConfig struct {
	NetworkMode string `json:"NetworkMode"`
}

type Mount struct {
	Name        string `json:"Name"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Driver      string `json:"Driver"`
	Mode        string `json:"Mode"`
	Rw          bool   `json:"RW"`
	Propagation string `json:"Propagation"`
}

type NetworkSettings struct {
	Networks Networks `json:"Networks"`
}

type Networks struct {
	Bridge Bridge `json:"bridge"`
}

type Bridge struct {
	NetworkID           string `json:"NetworkID"`
	EndpointID          string `json:"EndpointID"`
	Gateway             string `json:"Gateway"`
	IPAddress           string `json:"IPAddress"`
	IPPrefixLen         int64  `json:"IPPrefixLen"`
	IPv6Gateway         string `json:"IPv6Gateway"`
	GlobalIPv6Address   string `json:"GlobalIPv6Address"`
	GlobalIPv6PrefixLen int64  `json:"GlobalIPv6PrefixLen"`
	MACAddress          string `json:"MacAddress"`
}

type Port struct {
	PrivatePort int64  `json:"PrivatePort"`
	PublicPort  int64  `json:"PublicPort"`
	Type        string `json:"Type"`
}

func (h *Helper) GetContainers() ([]Container, error) {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _ string, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}

	url := "http://localhost/containers/json?all=1"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println("Unable to create get containers request", err)
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Unable to do get containers request", err)
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading get containers body", err)
		return nil, err
	}
	defer res.Body.Close()
	// fmt.Println(string(body))

	if res.StatusCode != http.StatusOK {
		fmt.Println("Invalid http status code in get containers", res.StatusCode)
		return nil, errors.New("Invalid http status code in get containers")
	}

	var containers []Container

	err = json.Unmarshal(body, &containers)
	if err != nil {
		fmt.Println("Error unmarshalling get containers body", err)
		return nil, err
	}

	return containers, nil
}

func (h *Helper) GetContianerLogs(containerId string) (io.ReadCloser, error) {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _ string, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
		Timeout: 0,
	}

	url := fmt.Sprintf(
		"http://localhost/containers/%s/logs?stdout=1&stderr=1&follow=1",
		containerId,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println("Unable to create get container logs request", err)
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Unable to do get container logs request", err)
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		// body, err := io.ReadAll(res.Body)
		// if err != nil {
		// 	fmt.Println("Error reading get containers body", err)
		// 	return nil, err
		// }
		// defer res.Body.Close()
		// fmt.Println(string(body))
		fmt.Println("Invalid http status code in get containers", res.StatusCode)
		return nil, errors.New("Invalid http status code in get containers")
	}

	return res.Body, nil
}
