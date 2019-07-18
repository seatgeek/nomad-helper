package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/Jeffail/tunny"
	"github.com/hashicorp/nomad/api"
	"github.com/olekukonko/tablewriter"
	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
	cli "gopkg.in/urfave/cli.v1"
)

type detailedNode struct {
	node   *api.Node
	allocs []*api.Allocation
}

func Run(c *cli.Context) error {
	dimensions := c.StringSlice("dimension")
	if len(dimensions) == 0 {
		log.Fatal("Please provide a list of dimensions")
	}

	nomadClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	clients, err := helpers.FilteredClientList(nomadClient, c)
	if err != nil {
		return err
	}

	pool := tunny.NewFunc(runtime.NumCPU()*2, func(payload interface{}) interface{} {
		log.Debugf("Reading client %s", payload.(*api.NodeListStub).ID)

		client, _, err := nomadClient.Nodes().Info(payload.(*api.NodeListStub).ID, &api.QueryOptions{AllowStale: true})
		if err != nil {
			panic(err)
		}

		// if client.Status != "ready" {
		// 	return nil
		// }

		res := detailedNode{}
		res.node = client
		// allocs, _, err := nomadClient.Nodes().Allocations(payload.(*api.NodeListStub).ID, nil)
		// if err != nil {
		// 	panic(err)
		// }
		// res.allocs = allocs
		log.Debugf("Done reading client %s", payload.(*api.NodeListStub).ID)
		return res
	})
	defer pool.Close()

	var wg sync.WaitGroup
	var l sync.Mutex
	detailedClients := make(map[string]detailedNode, 0)
	for _, client := range clients {
		wg.Add(1)

		go func(c *api.NodeListStub) {
			defer wg.Done()

			r := pool.Process(c)
			if r == nil {
				return
			}

			l.Lock()
			detailedClients[c.ID] = r.(detailedNode)
			l.Unlock()
		}(client)
	}
	wg.Wait()

	propReader := metaPropReader(dimensions...)
	result := computeStruct(detailedClients, propReader)

	switch c.String("output-format") {
	case "table":
		printTable(result, propReader)

	case "json":
		jsonText, err := json.Marshal(result)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(jsonText))

	case "json-pretty":
		jsonText, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(jsonText))

	default:
		return fmt.Errorf("Invalid output-format: %s", c.String("output-format"))
	}

	return nil
}

func computeStruct(nodes map[string]detailedNode, reader propReader) []*result {
	m := make([]*result, 0)

	for _, node := range nodes {
		names := reader.read(node)
		key := strings.Join(names, ".")

		if v, ok := get(m, key); ok {
			v.Value++
			continue
		}

		m = append(m, &result{
			Key:   key,
			Path:  names,
			Value: 1,
		})
	}

	sort.Slice(m, func(i, j int) bool {
		return m[i].Key < m[j].Key
	})

	return m
}

type result struct {
	Key   string   `json:"key"`
	Path  []string `json:"path"`
	Value int      `json:"value"`
}

func printTable(m []*result, reader propReader) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)

	header := reader.getKeys()
	header = append(header, "count")
	table.SetHeader(header)

	for i, l := range m {
		o := make([]string, len(l.Path))
		row := l.Path
		copy(o, l.Path)

		// hack: make sure the value of 'value' is never the same
		char := "\001"
		if i%2 == 0 {
			char = "\002"
		}

		row = append(row, fmt.Sprintf("%d%s", l.Value, char))
		table.Append(row)
	}

	table.Render()
}

func get(m []*result, name string) (*result, bool) {
	for _, x := range m {
		if (*x).Key == name {
			return x, true
		}
	}
	return nil, false
}

func metaPropReader(props ...string) reader {
	return reader{keys: props}
}

type reader struct {
	keys []string
}

func (r reader) getKeys() []string {
	return r.keys
}

func (r reader) read(node detailedNode) []string {
	s := make([]string, 0)

	for _, prop := range r.keys {
		chunks := strings.Split(prop, ".")
		switch chunks[0] {
		case "attributes", "Attributes":
			key := strings.Join(chunks[1:], ".")
			s = append(s, node.node.Attributes[key])

		case "meta", "Meta":
			key := strings.Join(chunks[1:], ".")
			s = append(s, node.node.Meta[key])

		case "NodeClass", "class":
			s = append(s, node.node.NodeClass)

		case "Name", "name":
			s = append(s, node.node.Name)

		case "Datacenter", "dc":
			s = append(s, node.node.Datacenter)

		case "Drain", "drain":
			s = append(s, fmt.Sprintf("%+v", node.node.Drain))

		case "Status", "status":
			s = append(s, node.node.Status)

		case "SchedulingEligibility":
			s = append(s, node.node.SchedulingEligibility)

		default:
			panic(fmt.Sprintf("don't know how to find value for '%s'", prop))
		}
	}

	return s
}

type propReader interface {
	read(node detailedNode) []string
	getKeys() []string
}
