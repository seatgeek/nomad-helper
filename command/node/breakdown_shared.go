package node

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/olekukonko/tablewriter"
	"github.com/seatgeek/nomad-helper/helpers"
)

func breakdownResponse(format string, nodes []*api.Node, propReader helpers.PropReader) (string, error) {
	// Compute result
	result, err := computeStruct(nodes, propReader)
	if err != nil {
		return "", err
	}

	// Create output based on requested format
	switch format {
	case "table":
		var b bytes.Buffer
		writer := bufio.NewWriter(&b)
		printTable(result, propReader, writer)
		writer.Flush()

		return b.String(), nil

	case "json":
		jsonText, err := json.Marshal(result)
		if err != nil {
			return "", err
		}

		return string(jsonText), nil

	case "json-pretty":
		jsonText, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}

		return string(jsonText), nil

	default:
		return "", fmt.Errorf("Invalid output-format: %s", format)
	}
}

func computeStruct(nodes []*api.Node, reader helpers.PropReader) ([]*result, error) {
	m := make([]*result, 0)

	for _, node := range nodes {
		names, err := reader.Read(node)
		if err != nil {
			return nil, err
		}

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

	return m, nil
}

type result struct {
	Key   string   `json:"key"`
	Path  []string `json:"path"`
	Value int      `json:"value"`
}

func printTable(m []*result, reader helpers.PropReader, writer io.Writer) {
	table := tablewriter.NewWriter(writer)
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
