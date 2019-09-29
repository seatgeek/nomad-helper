package node

import (
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

func get(m []*result, name string) (*result, bool) {
	for _, x := range m {
		if (*x).Key == name {
			return x, true
		}
	}
	return nil, false
}

func getCLIArgs(c *cli.Context) []string {
	input := deleteEmpty(append([]string{c.Args().First()}, c.Args().Tail()...))

	result := []string{}
	for _, key := range input {
		result = append(result, strings.Split(key, ",")...)
	}

	return input
}

func deleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func getData(c *cli.Context, logger *log.Logger) ([]*api.Node, error) {
	nomadClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	nodes, err := helpers.FilteredClientList(nomadClient, c.Parent(), logger)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}
