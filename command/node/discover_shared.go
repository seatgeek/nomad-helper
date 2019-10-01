package node

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
)

type DiscoverResponse struct {
	Meta      map[string][]string
	Attribute map[string][]string
	Node      map[string][]string
}

func discoverData(filters helpers.ClientFilter, logger *log.Logger, progress bool) (*DiscoverResponse, error) {
	// Collect Node data from the Nomad cluster
	nodes, err := getData(filters, logger, progress)
	if err != nil {
		return nil, err
	}

	nodeProperties := make(map[string][]string, 0)
	nodeProperties["class"] = make([]string, 0)
	nodeProperties["datacenter"] = make([]string, 0)
	nodeProperties["eligibility"] = make([]string, 0)
	nodeProperties["status"] = make([]string, 0)

	metaOptions := make(map[string][]string, 0)
	attributeOptions := make(map[string][]string, 0)

	for _, node := range nodes {
		for k, v := range node.Meta {
			if _, ok := metaOptions[k]; !ok {
				metaOptions[k] = make([]string, 0)
			}

			metaOptions[k] = AppendIfMissing(metaOptions[k], v)
		}

		for k, v := range node.Attributes {
			if strings.HasPrefix(k, "unique.") {
				continue
			}

			if _, ok := ignoredNodeAttributes[k]; ok {
				continue
			}

			if _, ok := attributeOptions[k]; !ok {
				attributeOptions[k] = make([]string, 0)
			}

			attributeOptions[k] = AppendIfMissing(attributeOptions[k], v)
		}

		nodeProperties["class"] = AppendIfMissing(nodeProperties["class"], node.NodeClass)
		nodeProperties["datacenter"] = AppendIfMissing(nodeProperties["datacenter"], node.Datacenter)
		nodeProperties["eligibility"] = AppendIfMissing(nodeProperties["eligibility"], node.SchedulingEligibility)
		nodeProperties["status"] = AppendIfMissing(nodeProperties["status"], node.Status)
	}

	resp := &DiscoverResponse{
		Attribute: attributeOptions,
		Meta:      metaOptions,
		Node:      nodeProperties,
	}

	return resp, nil
}

func discoverResponse(format string, input DiscoverResponse) (string, error) {
	// Create output based on requested format
	switch format {
	case "table":
		// Compute result
		result, err := computeDiscoverStruct(input)
		if err != nil {
			return "", err
		}

		propReader := helpers.NewMetaPropReader("Type", "Key", "Value")

		var b bytes.Buffer
		writer := bufio.NewWriter(&b)
		printDiscoverTable(result, propReader, writer)
		writer.Flush()
		return b.String(), nil

	case "json":
		jsonText, err := json.Marshal(input)
		if err != nil {
			return "", err
		}

		return string(jsonText), nil

	case "json-pretty":
		jsonText, err := json.MarshalIndent(input, "", "  ")
		if err != nil {
			return "", err
		}

		return string(jsonText), nil

	default:
		return "", fmt.Errorf("Invalid output-format: %s", format)
	}
}

func computeDiscoverStruct(result DiscoverResponse) ([][]string, error) {
	m := make([][]string, 0)

	for key, values := range result.Meta {
		for _, val := range values {
			m = append(m, []string{"--filter-meta", key, val})
		}
	}

	for key, values := range result.Attribute {
		for _, val := range values {
			m = append(m, []string{"--filter-attribute", key, val})
		}
	}

	for key, values := range result.Node {
		for _, val := range values {
			m = append(m, []string{"node", key, val})
		}
	}

	sort.Slice(m[:], func(i, j int) bool {
		for x := range m[i] {
			if m[i][x] == m[j][x] {
				continue
			}
			return m[i][x] < m[j][x]
		}
		return false
	})
	return m, nil
}

func printDiscoverTable(m [][]string, reader helpers.PropReader, writer io.Writer) {
	table := tablewriter.NewWriter(writer)
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	header := reader.GetKeys()
	table.SetHeader(header)
	table.AppendBulk(m)

	table.Render()
}
