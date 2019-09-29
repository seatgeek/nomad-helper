package node

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/nomad/api"
	"github.com/olekukonko/tablewriter"
	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

func List(c *cli.Context) error {
	// Default list of fields if none are provided by the user
	fields := []string{"name", "status", "SchedulingEligibility", "drain", "class"}

	// Allow user to provide list of fields
	if input := getCLIArgs(c); len(input) > 0 {
		fields = input
	}

	// Collect Node data from the Nomad cluster
	nodes, err := getData(c)
	if err != nil {
		log.Fatal(err)
	}

	// Create a prop reader for results
	propReader := helpers.NewMetaPropReader(fields...)

	// Create output based on requested format
	switch c.String("output-format") {
	case "table":
		printListTable(computeListTableStruct(nodes, propReader), propReader)

	case "json":
		jsonText, err := json.Marshal(computeListRawStruct(nodes, propReader))
		if err != nil {
			panic(err)
		}
		fmt.Println(string(jsonText))

	case "json-pretty":
		jsonText, err := json.MarshalIndent(computeListRawStruct(nodes, propReader), "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(jsonText))

	default:
		return fmt.Errorf("Invalid output-format: %s", c.String("output-format"))
	}

	return nil
}

func computeListTableStruct(nodes map[string]*api.Node, reader helpers.PropReader) [][]string {
	m := make([][]string, 0)

	for _, node := range nodes {
		names := reader.Read(node)
		m = append(m, names)
	}

	return m
}

func computeListRawStruct(nodes map[string]*api.Node, reader helpers.PropReader) []map[string]string {
	m := make([]map[string]string, 0)

	for _, node := range nodes {
		m = append(m, reader.ReadMap(node))
	}

	return m
}

func printListTable(m [][]string, reader helpers.PropReader) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoMergeCells(false)
	table.SetRowLine(true)

	header := reader.GetKeys()
	table.SetHeader(header)
	table.AppendBulk(m)

	table.Render()
}
