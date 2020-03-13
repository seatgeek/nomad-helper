package node

import (
	"fmt"

	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

func EmptyCLI(c *cli.Context, logger *log.Logger) error {
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

	emptyNodes, err := filterForEmpty(nodes)
	if err != nil {
		return err
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
