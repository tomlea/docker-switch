package main

import (
	"context"
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/docker/docker/api/types"
	//"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"log"
	"time"
)

import "flag"

func main() {
	// create an accessory
	namePtr := flag.String("switch", "", "Name of switch")
	flag.Parse()
	container := flag.Arg(0)

	if *namePtr == "" {
		namePtr = &container
	}
	log.Println(*namePtr)

	if *namePtr == "" {
		panic("Required argument: contaner name")
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	info := accessory.Info{Name: *namePtr}
	ac := accessory.NewSwitch(info)

	// configure the ip transport
	t, err := hc.NewIPTransport(hc.Config{Pin: "18927133"}, ac.Accessory)
	if err != nil {
		log.Panic(err)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	sw := Switchable{
		container: &container,
		cli:       cli,
		ac:        ac,
	}

  sw.monitor()

	t.Start()
}

type Switchable struct {
	container *string
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
	log.Println("Stopping container")
	sw.cli.ContainerStop(context.Background(), *sw.container, nil)
}
func (sw *Switchable) startContainer() {
	log.Println("Starting container")
	sw.cli.ContainerStart(context.Background(), *sw.container, types.ContainerStartOptions{})
}

func (sw *Switchable) Poll() {
	containerStats, err := sw.cli.ContainerInspect(context.Background(), *sw.container)
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
