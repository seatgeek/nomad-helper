package node

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

func Stats(c *cli.Context) error {
	dimensions := c.StringSlice("dimension")
	if len(dimensions) == 0 {
		log.Fatal("Please provide a list of dimensions")
	}

	args := make([]string, 0)
	for _, dimension := range dimensions {
		args = append(args, dimension)
	}

	flags := ""

	if value := c.String("filter-prefix"); len(value) > 0 {
		flags = flags + fmt.Sprintf("--filter-prefix=%+v ", value)
	}

	if value := c.String("filter-class"); len(value) > 0 {
		flags = flags + fmt.Sprintf("--filter-class=%+v ", value)
	}

	if value := c.String("filter-version"); len(value) > 0 {
		flags = flags + fmt.Sprintf("--filter-version=%+v ", value)
	}

	if value := c.String("filter-eligibility"); len(value) > 0 {
		flags = flags + fmt.Sprintf("--filter-eligibility=%+v ", value)
	}

	if value := c.Int("percent"); value != 100 {
		flags = flags + fmt.Sprintf("--percent=%+v ", value)
	}

	if value := c.StringSlice("filter-meta"); len(value) > 0 {
		for _, item := range value {
			flags = flags + fmt.Sprintf("--filter-meta=%+v ", item)
		}
	}

	if value := c.StringSlice("filter-attribute"); len(value) > 0 {
		for _, item := range value {
			flags = flags + fmt.Sprintf("--filter-attribute=%+v ", item)
		}
	}

	return fmt.Errorf("nomad-helper node %sbreakdown %s", flags, strings.Join(args, " "))

	// return nil
}
