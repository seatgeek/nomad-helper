package node

import (
	"fmt"
	"io"

	"github.com/hashicorp/nomad/api"
	"github.com/olekukonko/tablewriter"
	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

func ListCLI(c *cli.Context, logger *log.Logger) error {
	// Default list of fields if none are provided by the user
	fields := []string{"name", "status", "SchedulingEligibility", "drain", "class"}

	// Allow user to provide list of fields
	if input := getCLIArgs(c); len(input) > 0 {
		fields = input
	}

	filters := helpers.ClientFilterFromCLI(c)

	// Collect Node data from the Nomad cluster
	nodes, err := getData(filters, logger, !c.BoolT("no-progress"))
	if err != nil {
		log.Fatal(err)
	}

	// Create a prop reader for results
	propReader := helpers.NewMetaPropReader(fields...)

	res, err := listResponse(c.String("output-format"), nodes, propReader)
	if err != nil {
		return err
	}

	fmt.Println(res)
	return nil
}

func computeListTableStruct(nodes []*api.Node, reader helpers.PropReader) ([][]string, error) {
	m := make([][]string, 0)

	for _, node := range nodes {
		names, err := reader.Read(node)
		if err != nil {
			return nil, err
		}

		m = append(m, names)
	}

	return m, nil
}

func printListTable(m [][]string, reader helpers.PropReader, writer io.Writer) {
	table := tablewriter.NewWriter(writer)
	table.SetAutoMergeCells(false)
	table.SetRowLine(true)

	header := reader.GetKeys()
	table.SetHeader(header)
	table.AppendBulk(m)

	table.Render()
}
