package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	networkName     = "my-routed-network"
	domainSuffix    = ".local"
	containerPrefix = "container-"
)

func publishARecord(containerName, ipAddress string) {
	fqdn := fmt.Sprintf("%s%s%s", containerPrefix, containerName, domainSuffix)
	cmd := exec.Command("go-avahi-cname", "cname", "--fqdn", fqdn, ipAddress)
	if err := cmd.Run(); err != nil {
		log.Printf("Error publishing A record: %v", err)
	}
}

func publishCNAME(containerName, friendlyName string) {
	fqdn := fmt.Sprintf("%s%s%s", containerPrefix, containerName, domainSuffix)
	cname := fmt.Sprintf("%s%s", friendlyName, domainSuffix)
	cmd := exec.Command("go-avahi-cname", "cname", "--fqdn", fqdn, cname)
	if err := cmd.Run(); err != nil {
		log.Printf("Error publishing CNAME: %v", err)
	}
}

func handleContainerEvent(cli *client.Client, event types.EventMessage) {
	if event.Type == "network" && event.Action == "connect" && event.Actor.Attributes["name"] == networkName {
		containerID := event.Actor.Attributes["container"]
		container, err := cli.ContainerInspect(context.Background(), containerID)
		if err != nil {
			log.Printf("Error inspecting container: %v", err)
			return
		}

		containerName := container.Name
		ipAddress := container.NetworkSettings.Networks[networkName].IPAddress

		// Publish A record
		publishARecord(containerName, ipAddress)

		// Publish CNAME (you might want to derive the friendly name from container labels or some other logic)
		friendlyName := fmt.Sprintf("service-%s", containerName)
		publishCNAME(containerName, friendlyName)
	}
}

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error creating Docker client: %v", err)
	}

	eventsChan, errChan := cli.Events(context.Background(), types.EventsOptions{})

	for {
		select {
		case event := <-eventsChan:
			handleContainerEvent(cli, event)
		case err := <-errChan:
			log.Printf("Error from events channel: %v", err)
		}
	}
}
