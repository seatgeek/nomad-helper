package node

import (
	"fmt"

	"github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

var emptyDefaultFields = []string{"name", "status", "SchedulingEligibility", "drain", "class"}

type jobTypes struct {
	system  int
	service int
	batch   int
}

type nodeList map[string]jobTypes

func Empty(c *cli.Context, logger *log.Logger) error {
	nomadClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	// Default list of fields if none are provided by the user
	fields := emptyDefaultFields

	// Allow user to provide list of fields
	if input := getCLIArgs(c); len(input) > 0 {
		fields = input
	}

	filters := helpers.ClientFilterFromCLI(c.Parent())

	// Collect Node data from the Nomad cluster
	nodes, err := getData(filters, logger, !c.BoolT("no-progress"))
	if err != nil {
		log.Fatal(err)
	}

	// Construct list of Node IDs
	nodeIDs := make(map[string]*api.Node)
	for _, node := range nodes {
		nodeIDs[node.ID] = node
	}

	log.Infof("Reading cluster allocations...")
	allocations, _, err := nomadClient.Allocations().List(nil)
	if err != nil {
		return err
	}

	nodesByType := nodeList{}
	for _, allocation := range allocations {
		if _, ok := nodeIDs[allocation.NodeID]; !ok {
			continue
		}

		if allocation.ClientStatus != "running" {
			continue
		}

		node := nodesByType[allocation.NodeID]

		switch allocation.JobType {
		case "system":
			node.system++
		case "batch":
			node.batch++
		case "service":
			node.service++
		}

		nodesByType[allocation.NodeID] = node
	}

	emptyNodes := make([]*api.Node, 0)
	for nodeID, stats := range nodesByType {
		if stats.batch == 0 && stats.service == 0 {
			emptyNodes = append(emptyNodes, nodeIDs[nodeID])
		}
	}

	if len(emptyNodes) == 0 {
		return fmt.Errorf("Found no empty nodes")
	}

	// Create a prop reader for results
	propReader := helpers.NewMetaPropReader(fields...)

	res, err := listResponse(c.String("output-format"), emptyNodes, propReader)
	if err != nil {
		return err
	}

	fmt.Println(res)
	return nil
}

func filterForEmpty(nodes []*api.Node) ([]*api.Node, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	// Construct list of Node IDs
	nodeIDs := make(map[string]*api.Node)
	for _, node := range nodes {
		nodeIDs[node.ID] = node
	}

	log.Infof("Reading cluster allocations...")
	allocations, _, err := client.Allocations().List(nil)
	if err != nil {
		return nil, err
	}

	nodesByType := nodeList{}
	for _, allocation := range allocations {
		if _, ok := nodeIDs[allocation.NodeID]; !ok {
			continue
		}

		if allocation.ClientStatus != "running" {
			continue
		}

		node := nodesByType[allocation.NodeID]

		switch allocation.JobType {
		case "system":
			node.system++
		case "batch":
			node.batch++
		case "service":
			node.service++
		}

		nodesByType[allocation.NodeID] = node
	}

	emptyNodes := make([]*api.Node, 0)
	for nodeID, stats := range nodesByType {
		if stats.batch == 0 && stats.service == 0 {
			emptyNodes = append(emptyNodes, nodeIDs[nodeID])
		}
	}

	return emptyNodes, nil
}
