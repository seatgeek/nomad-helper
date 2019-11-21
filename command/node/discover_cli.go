package node

import (
	"fmt"

	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

var ignoredNodeAttributes = map[string]interface{}{
	"cpu.frequency":     nil,
	"cpu.modelname":     nil,
	"cpu.totalcompute":  nil,
	"memory.totalbytes": nil,
	"os.signals":        nil,
}

func DiscoverCLI(c *cli.Context, logger *log.Logger) error {
	filters := helpers.ClientFilterFromCLI(c.Parent())

	data, err := discoverData(filters, logger, !c.BoolT("no-progress"))
	if err != nil {
		return err
	}

	output, err := discoverResponse(c.String("output-format"), *data)
	if err != nil {
		return err
	}

	fmt.Println(output)
	return nil
}

func AppendIfMissing(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}

	return append(slice, i)
}
