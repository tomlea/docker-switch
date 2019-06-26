package main

import (
	"context"
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pelletier/go-toml"
	"log"
	"time"
)

type ContainerSwitch struct {
	Accessory     *accessory.Switch
	ContainerName string
}

func create_container_switch(name string, container string) *ContainerSwitch {
	ac := accessory.NewSwitch(accessory.Info{Name: name})
	log.Printf("name = %s, container = %s", name, container)
	return &ContainerSwitch{
		Accessory:     ac,
		ContainerName: container,
	}
}

func (cs *ContainerSwitch) monitor(docker_cli *client.Client) {
	sw := Switchable{
		container: cs.ContainerName,
		cli:       docker_cli,
		ac:        cs.Accessory,
	}
	sw.monitor()
}

func parse_config(path string) []*ContainerSwitch {
	servers_config, err := toml.LoadFile(path)
	if err != nil {
		log.Panic(err)
	}
	config_keys := servers_config.Keys()
	container_switches := make([]*ContainerSwitch, len(config_keys))
	for i := range config_keys {
		key := config_keys[i]
		container_config := servers_config.Get(key).(*toml.Tree)
		container := container_config.GetDefault("container", key).(string)
		name := container_config.GetDefault("name", key).(string)
		container_switches[i] = create_container_switch(name, container)
	}

	return container_switches
}

func container_switches_to_accessories(container_switches []*ContainerSwitch) []*accessory.Accessory {
	accessories := make([]*accessory.Accessory, len(container_switches))

	for i := range container_switches {
		accessories[i] = container_switches[i].Accessory.Accessory
	}
	return accessories
}

func main() {
	docker_cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	container_switches := parse_config("config.toml")
	accessories := container_switches_to_accessories(container_switches)
	for i := range container_switches {
		container_switches[i].monitor(docker_cli)
	}
	bridge := accessory.NewBridge(accessory.Info{Name: "Docker Switch Bridge"}).Accessory
	transport, err := hc.NewIPTransport(hc.Config{Pin: "18927133", StoragePath: "state"}, bridge, accessories...)
	if err != nil {
		log.Panic(err)
	}
	hc.OnTermination(func() {
		<-transport.Stop()
	})
	transport.Start()
}

type Switchable struct {
	container string
	cli       *client.Client
	ac        *accessory.Switch
}

func (sw *Switchable) monitor() {
	sw.ac.Switch.On.OnValueRemoteUpdate(func(on bool) {
		if on == true {
			sw.startContainer()
		} else {
			sw.stopContainer()
		}
	})
	go sw.PollContinuously()
}

func (sw *Switchable) stopContainer() {
	log.Printf("Stopping container %s", sw.container)
	sw.cli.ContainerStop(context.Background(), sw.container, nil)
}

func (sw *Switchable) startContainer() {
	log.Printf("Starting container %s", sw.container)
	sw.cli.ContainerStart(context.Background(), sw.container, types.ContainerStartOptions{})
}

func (sw *Switchable) Poll() {
	containerStats, err := sw.cli.ContainerInspect(context.Background(), sw.container)
	if err != nil {
		log.Println(err)
	} else {
		log.Printf("Container running = %t\n", containerStats.State.Running)
		sw.ac.Switch.On.SetValue(containerStats.State.Running)
	}
}

func (sw *Switchable) PollContinuously() {
	for {
		sw.Poll()
		time.Sleep(10 * time.Second)
	}
}
