package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	dockerTypes "github.com/docker/docker/api/types"
	containerTypes "github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
)

func restartContainer(cli *dockerClient.Client, container dockerTypes.Container) error {
	return cli.ContainerRestart(context.Background(), container.ID, containerTypes.StopOptions{})
}

func getContainerWithLabel(cli *dockerClient.Client, label string) (dockerTypes.Container, error) {
	var desiredContainer dockerTypes.Container
	var found = false

	containers, err := cli.ContainerList(context.Background(), dockerTypes.ContainerListOptions{})
	if err != nil {
		return dockerTypes.Container{}, err
	}

	for _, container := range containers {
		for lblName := range container.Labels {
			if lblName == label {
				if found {
					return dockerTypes.Container{},
						fmt.Errorf("multiple containers with label %s found", label)
				}
				desiredContainer = container
				found = true
			}
		}
	}

	if !found {
		return dockerTypes.Container{},
			fmt.Errorf("no containers with label %s found", label)
	}

	return desiredContainer, nil
}

/*
input:

<------ fullLabelName ------->
thats.prefix.objName1.keyName1 = labelValue1
thats.prefix.objName2.keyName1 = labelValue2
thats.prefix.objName2.keyName2 = 2
<- prefix ->

output:

	{
		"objName1": {
			"keyName1": "labelValue1"
		},
		"objName2": {
			"keyName1": "labelValue2",
			"keyName2": 2,
		}
	}
*/
func labelsToStructs[T any](prefix string, container dockerTypes.Container) (map[string]T, error) {
	var objects = make(map[string]T)
	fields := reflect.VisibleFields(reflect.TypeOf(objects).Elem())

	for fullLabelName, labelValue := range container.Labels {
		labelName := strings.TrimPrefix(fullLabelName, prefix+".")
		if labelName == fullLabelName {
			continue
		}

		s := strings.Split(labelName, ".")
		if len(s) != 2 {
			log.Printf("unable to parse label %s, skipping", fullLabelName)
			continue
		}
		objName, keyName := s[0], s[1]

		object, ok := objects[objName]
		if !ok {
			setFieldDefaults(&object)
		}
		for _, field := range fields {
			fieldName := field.Tag.Get("label")
			if fieldName == "" {
				continue
			}
			if fieldName == keyName {
				setField(&object, field.Name, labelValue)
				objects[objName] = object
				continue
			}
		}
	}

	return objects, nil
}
