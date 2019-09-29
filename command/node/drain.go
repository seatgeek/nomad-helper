package node

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

func Drain(c *cli.Context) error {
	// Check that enable or disable is not set with monitor
	if c.Bool("monitor") && (c.Bool("enable") || c.Bool("disable")) {
		return fmt.Errorf("The -monitor flag cannot be used with the '-enable' or '-disable' flags")
	}

	// Check that we got either enable or disable, but not both.
	if (c.Bool("enable") && c.Bool("disable")) || (!c.Bool("monitor") && !c.Bool("enable") && !c.Bool("disable")) {
		return fmt.Errorf("Ethier the '-enable' or '-disable' flag must be set, unless using '-monitor'")
	}

	// Validate a compatible set of flags were set
	if c.Bool("disable") && (c.Bool("force") || c.Bool("no-deadline") || c.Bool("ignore-system")) {
		return fmt.Errorf("-disable can't be combined with flags configuring drain strategy")
	}
	if c.Bool("force") && c.Bool("no-deadline") {
		return fmt.Errorf("-force and -no-deadline are mutually exclusive")
	}

	deadline := c.Duration("deadline")
	if c.Bool("force") {
		deadline = -1
	}
	if c.Bool("no-deadline") {
		deadline = 0
	}

	// create Nomad API client
	nomadClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	// find nodes to target
	matches, err := helpers.FilteredClientList(nomadClient, c.Parent())
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		return fmt.Errorf("Could not find any nodes matching provided filters")
	}

	var wg sync.WaitGroup
	ctx := context.Background()

	for _, node := range matches {
		log.Infof("Node %s (class: %s / version: %s)", node.Name, node.NodeClass, node.Version)

		// in monitor mode we don't do any change to node state
		if c.Bool("monitor") {
			go monitor(ctx, nomadClient, node, &wg)
			continue
		}

		var spec *api.DrainSpec
		if c.Bool("enable") {
			spec = &api.DrainSpec{
				Deadline:         deadline,
				IgnoreSystemJobs: c.Bool("ignore-system"),
			}
		}

		_, err := nomadClient.Nodes().UpdateDrain(node.ID, spec, !c.Bool("keep-ineligible"), nil)
		if err != nil {
			log.Errorf("Could not update drain config for %s: %s", node.Name, err)
			continue
		}

		if !c.Bool("enable") || c.Bool("detach") {
			if c.Bool("enable") {
				log.Infof("Node %q drain strategy set", node.ID)
			} else {
				log.Infof("Node %q drain strategy unset", node.ID)
			}
		}

		if c.Bool("enable") && !c.Bool("detach") {
			go monitor(ctx, nomadClient, node, &wg)
		}
	}

	wg.Wait()

	return nil
}

func monitor(ctx context.Context, client *api.Client, node *api.NodeListStub, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	logger := log.WithField("node", node.Name)
	ch := client.Nodes().MonitorDrain(ctx, node.ID, 0, false)
	for {
		select {
		case m, ok := <-ch:
			if !ok { // channel closed
				return
			}

			switch m.Level {
			case api.MonitorMsgLevelNormal:
				logger.Info(m.String())

			case api.MonitorMsgLevelInfo:
				logger.Info(m.String())

			case api.MonitorMsgLevelWarn:
				logger.Warn(m.String())

			case api.MonitorMsgLevelError:
				logger.Error(m.String())
			}

		}
	}
}
