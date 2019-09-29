package node

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/olekukonko/tablewriter"
	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

func Breakdown(c *cli.Context, logger *log.Logger) error {
	// Get list of CLI arguments we should use as dimensions
	dimensions := getCLIArgs(c)

	// We require at minimum one field
	if len(dimensions) == 0 {
		logger.Fatal("Missing argument for list of fields to use as dimensions")
	}

	// Collect Node data from the Nomad cluster
	nodes, err := getData(c, logger)
	if err != nil {
		logger.Fatal(err)
	}

	// Create a prop reader for results
	propReader := helpers.NewMetaPropReader(dimensions...)

	// Compute result
	result := computeStruct(nodes, propReader)

	// Create output based on requested format
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

func computeStruct(nodes []*api.Node, reader helpers.PropReader) []*result {
	m := make([]*result, 0)

	for _, node := range nodes {
		names := reader.Read(node)
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

func printTable(m []*result, reader helpers.PropReader) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)

	header := reader.GetKeys()
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
