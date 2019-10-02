package node

import (
	"fmt"

	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

func BreakdownCLI(c *cli.Context, logger *log.Logger) error {
	// Get list of CLI arguments we should use as dimensions
	dimensions := getCLIArgs(c)

	// We require at minimum one field
	if len(dimensions) == 0 {
		logger.Fatal("Missing argument for list of fields to use as dimensions")
	}

	filters := helpers.ClientFilterFromCLI(c)

	// Collect Node data from the Nomad cluster
	nodes, err := getData(filters, logger, !c.BoolT("no-progress"))
	if err != nil {
		return err
	}

	// Create a prop reader for results
	propReader := helpers.NewMetaPropReader(dimensions...)

	// Output result
	output, err := breakdownResponse(c.String("output-format"), nodes, propReader)
	if err != nil {
		return err
	}

	fmt.Println(output)
	return nil
}
