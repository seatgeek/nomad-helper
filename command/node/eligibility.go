package node

import (
	"fmt"

	"github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

func Eligibility(c *cli.Context, logger *log.Logger) error {
	// Check that we got either enable or disable, but not both.
	if (c.Bool("enable") && c.Bool("disable")) || (!c.Bool("enable") && !c.Bool("disable")) {
		return fmt.Errorf("Ethier the '-enable' or '-disable' flag must be set")
	}

	nomadClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	matches, err := helpers.FilteredClientList(nomadClient, c.Parent(), logger)
	if err != nil {
		return err
	}

	for _, node := range matches {
		log.Infof("Node %s (class: %s / version: %s)", node.Name, node.NodeClass, node.Attributes["nomad.version"])

		_, err := nomadClient.Nodes().ToggleEligibility(node.ID, c.Bool("enable"), nil)
		if err != nil {
			log.Errorf("Error updating scheduling eligibility for %s: %s", node.Name, err)
			continue
		}

		if c.Bool("enable") {
			log.Infof("Node %q scheduling eligibility set: eligible for scheduling", node.ID)
		} else {
			log.Infof("Node %q scheduling eligibility set: ineligible for scheduling", node.ID)
		}
	}

	return nil
}
