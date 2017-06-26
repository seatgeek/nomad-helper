package main

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/nomad/api"
)

func drainCommand() error {
	log.Info("Starting drain")

	// Create Nomad API client
	client, err := NewNomadClient()
	if err != nil {
		return err
	}

	// Read current agents info
	self, err := client.Agent().Self()
	if err != nil {
		return err
	}

	nodeID, ok := self.Stats["client"]["node_id"]
	if !ok {
		return fmt.Errorf("Could not find client node id")
	}

	// Start drain mode
	_, err = client.Nodes().ToggleDrain(nodeID, true, &api.WriteOptions{})
	if err != nil {
		return err
	}

	// Wait for allocations to drain
	index := uint64(0)

	for {
		options := &api.QueryOptions{
			WaitIndex: index,
			WaitTime:  time.Second * 30,
		}

		allocations, meta, err := client.Nodes().Allocations(nodeID, options)
		if err != nil {
			return nil
		}

		index = meta.LastIndex

		pending := 0

		for _, alloc := range allocations {
			switch alloc.ClientStatus {
			case "running":
				pending = pending + 1
			case "pending":
				pending = pending + 1
			}
		}

		if pending == 0 {
			break
		}

		log.Infof("%d pending allocations left", pending)
	}

	log.Info("Done!")

	return nil
}
