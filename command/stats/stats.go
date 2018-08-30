package stats

import (
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

	distributeRunningAllocsOn("", detailedClients, metaPropReader(dimensions...))

	// distributeRunningAllocsOn("NodeClass", detailedClients, metaPropReader("NodeClass"))
	// distributeRunningAllocsOn("AZ + Class", detailedClients, metaPropReader("meta.aws.instance.availability-zone", "NodeClass"))
	// distributeRunningAllocsOn("Instance type", detailedClients, metaPropReader("meta.aws.instance.availability-zone", "meta.aws.instance.type"))
	// distributeRunningAllocsOn("Instance type", detailedClients, metaPropReader("meta.aws.instance.life-cycle", "meta.aws.instance.availability-zone", "meta.aws.instance.type"))

	// spew.Dump(detailedClients)
	return nil
}

type hit struct {
	name  string
	names []string
	value int
}

func distributeRunningAllocsOn(name string, nodes map[string]detailedNode, reader propReader) {
	m := make([]*hit, 0)

	for _, node := range nodes {
		names := reader.read(node)
		key := strings.Join(names, ".")

		if v, ok := get(m, key); ok {
			v.value++
			continue
		}

		m = append(m, &hit{
			name:  key,
			names: names,
			value: 1,
		})
	}

	sort.Slice(m, func(i, j int) bool {
		return m[i].name < m[j].name
	})

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)

	header := reader.getKeys()
	header = append(header, "count")
	table.SetHeader(header)

	for i, l := range m {
		o := make([]string, len(l.names))
		row := l.names
		copy(o, l.names)

		// hack: make sure the value of 'value' is never the same
		char := "\001"
		if i%2 == 0 {
			char = "\002"
		}

		row = append(row, fmt.Sprintf("%d%s", l.value, char))
		table.Append(row)
	}

	table.Render()
}

func get(m []*hit, name string) (*hit, bool) {
	for _, x := range m {
		if (*x).name == name {
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
