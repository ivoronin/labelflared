package main

import (
	"context"
	_ "embed"
	"log"
	"path/filepath"

	dockerTypes "github.com/docker/docker/api/types"
	eventTypes "github.com/docker/docker/api/types/events"
	dockerClient "github.com/docker/docker/client"
)

const DefaultLabelPrefix = "labelflared"
const DefaultConfigDir = "/etc/cloudflared"

type Options struct {
	configPath  string
	credsPath   string
	tunnelUUID  string
	labelPrefix string
}

func restartCloudflared(cli *dockerClient.Client, labelPrefix string) {
	containerLabel := labelPrefix + ".cloudflared"
	container, err := getContainerWithLabel(cli, containerLabel)
	if err != nil {
		log.Printf("unable to find cloudflared container: %v", err)
		return
	}
	containerName := getContainerName(container)

	log.Printf("restarting cloudflared container %s", containerName)
	err = restartContainer(cli, container)
	if err != nil {
		log.Printf("unable to restart cloudflared container %s: %v", containerName, err)
	}
}

func refresh(cli *dockerClient.Client, options Options) {
	log.Printf("refreshing cloudflared configuration")

	config, err := renderConfig(cli, options)
	if err != nil {
		log.Printf("unable to generate cloudflared config: %v", err)
		return
	}

	hasChanged, err := writeConfigIfChanged(options.configPath, config)
	if err != nil {
		log.Printf("unable to write cloudflared config: %v", err)
		return
	}

	if hasChanged {
		log.Printf("configuration change detected")
		restartCloudflared(cli, options.labelPrefix)
	} else {
		log.Printf("no configuration change detected")
	}
}

func main() {
	var options Options

	configDir := defaultEnv("CLOUDFLARED_CONFIG_DIR", DefaultConfigDir)
	options.configPath = filepath.Join(configDir, "config.yml")
	options.credsPath = filepath.Join(configDir, "credentials.json")

	options.labelPrefix = defaultEnv("LABEL_PREFIX", DefaultLabelPrefix)

	b64EncodedToken := requireEnv("CLOUDFLARED_TOKEN")

	token, err := parseB64EncodedToken(b64EncodedToken)
	if err != nil {
		log.Fatalf("token parse error: %s", err)
	}

	options.tunnelUUID = token.TunnelID

	err = writeCredentialsFile(options.credsPath, token)
	if err != nil {
		log.Fatalf("unable to write credentials file: %s", err)
	}

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
		case msg := <-messages:
			if msg.Type == eventTypes.ContainerEventType &&
				(msg.Action == "create" || msg.Action == "destroy") {
				refresh(cli, options)
			}
		}
	}
}
