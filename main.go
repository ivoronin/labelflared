package main

import (
	"context"
	_ "embed"
	"log"

	dockerTypes "github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
)

const DefaultLabelPrefix = "labelflared"

type Options struct {
	configPath  string
	credsPath   string
	tunnelUUID  string
	labelPrefix string
}

func refresh(cli *dockerClient.Client, options Options) {
	log.Printf("refreshing cloudflared configuration")

	cfdContainerLabel := options.labelPrefix + ".cloudflared"
	cfdContainer, err := getContainerWithLabel(cli, cfdContainerLabel)
	if err != nil {
		log.Printf("unable to find cloudflared container: %v", err)
		return
	}

	cfdConfig, err := renderConfig(cli, options)
	if err != nil {
		log.Printf("unable to generate cloudflared config: %v", err)
		return
	}

	hasChanged, err := writeConfigIfChanged(options.configPath, cfdConfig)
	if err != nil {
		log.Printf("unable to write cloudflared config: %v", err)
		return
	}

	if hasChanged {
		log.Printf("configuration change detected, restarting container %s", cfdContainer.Names[0])
		err = restartContainer(cli, cfdContainer)
		if err != nil {
			log.Printf("unable to restart cloudflared container: %v", err)
		}
	} else {
		log.Printf("no configuration change detected")
	}
}

func main() {
	var options Options

	options.configPath = requireEnv("CONFIG_PATH")
	options.tunnelUUID = requireEnv("TUNNEL_UUID")
	options.credsPath = requireEnv("CREDENTIALS_FILE")
	options.labelPrefix = defaultEnv("LABEL_PREFIX", DefaultLabelPrefix)

	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv)
	if err != nil {
		log.Fatalf("docker client error: %s", err)
	}
	cli.NegotiateAPIVersion(context.Background())

	log.Print("labelflared started")

	refresh(cli, options)

	messages, errors := cli.Events(context.Background(), dockerTypes.EventsOptions{})
	for {
		select {
		case err := <-errors:
			log.Fatalf("error receiving event: %s", err)
		case <-messages:
			refresh(cli, options)
		}
	}
}