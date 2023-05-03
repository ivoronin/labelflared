package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

//go:embed config.go.yml
var ConfigTemplateString string
var ConfigTemplate = template.Must(template.New("Config").Parse(ConfigTemplateString))

type ConfigData struct {
	TunnelUUID   string
	CredsPath    string
	IngressRules []IngressRule
}

type IngressRule struct {
	Name     string
	Priority int    `label:"priority" default:"0"`
	Hostname string `label:"hostname"`
	Path     string `label:"path"`
	Protocol string `label:"protocol" default:"http"`
	Service  string
	Port     int `label:"port" default:"80"`
}

func validateContainerRules(containerRules map[string]IngressRule) error {
	for name, rule := range containerRules {
		if rule.Port == 0 {
			return fmt.Errorf("rule %s must have a port set", name)
		}
		if rule.Hostname == "" && rule.Path == "" {
			return fmt.Errorf("rule %s must have hostname or path set", name)
		}
	}

	return nil
}

func renderConfig(cli *client.Client, options Options) ([]byte, error) {
	var configBuf bytes.Buffer
	var rules []IngressRule
	var ingressRuleLabelPrefix = options.labelPrefix + ".ingress"

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		containerRules, err := labelsToStructs[IngressRule](ingressRuleLabelPrefix, container)
		if err != nil {
			return nil, err
		}
		if validateContainerRules(containerRules) != nil {
			return nil, err
		}

		/* Populate fields */
		for name, rule := range containerRules {
			rule.Name = name
			serviceHostname := strings.TrimPrefix(container.Names[0], "/")
			serviceURL := url.URL{
				Scheme: rule.Protocol,
				Host:   fmt.Sprintf("%s:%d", serviceHostname, rule.Port),
			}
			rule.Service = serviceURL.String()
			rules = append(rules, rule)
		}
	}

	sort.SliceStable(rules[:], func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	config := ConfigData{
		TunnelUUID:   options.tunnelUUID,
		CredsPath:    options.credsPath,
		IngressRules: rules,
	}

	err = ConfigTemplate.Execute(&configBuf, config)
	if err != nil {
		return nil, err
	}

	return configBuf.Bytes(), nil
}

func writeConfigIfChanged(configPath string, newConfig []byte) (bool, error) {
	configFile, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return false, err
	}
	defer configFile.Close()

	oldConfigSum := sha256.New()
	if _, err := io.Copy(oldConfigSum, configFile); err != nil {
		return false, err
	}

	newConfigSum := sha256.New()
	newConfigSum.Write(newConfig)

	if bytes.Equal(oldConfigSum.Sum(nil), newConfigSum.Sum(nil)) {
		return false, nil
	}

	err = configFile.Truncate(0)
	if err != nil {
		return false, err
	}
	_, err = configFile.Seek(0, 0)
	if err != nil {
		return false, err
	}
	_, err = configFile.Write(newConfig)
	if err != nil {
		return false, err
	}

	return true, nil
}
