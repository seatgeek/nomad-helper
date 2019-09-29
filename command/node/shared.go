package node

import (
	"runtime"
	"strings"
	"sync"

	"github.com/Jeffail/tunny"
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

func getData(c *cli.Context) (map[string]*api.Node, error) {
	nomadClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	clients, err := helpers.FilteredClientList(nomadClient, c.Parent())
	if err != nil {
		return nil, err
	}

	pool := tunny.NewFunc(runtime.NumCPU()*2, func(payload interface{}) interface{} {
		log.Debugf("Reading client %s", payload.(*api.NodeListStub).ID)

		node, _, err := nomadClient.Nodes().Info(payload.(*api.NodeListStub).ID, &api.QueryOptions{AllowStale: true})
		if err != nil {
			panic(err)
		}

		log.Debugf("Done reading client %s", node.ID)
		return node
	})
	defer pool.Close()

	var wg sync.WaitGroup
	var l sync.Mutex
	nodes := make(map[string]*api.Node, 0)

	for _, client := range clients {
		wg.Add(1)

		go func(c *api.NodeListStub) {
			defer wg.Done()

			r := pool.Process(c)
			if r == nil {
				return
			}

			l.Lock()
			nodes[c.ID] = r.(*api.Node)
			l.Unlock()
		}(client)
	}
	wg.Wait()

	return nodes, nil
}
