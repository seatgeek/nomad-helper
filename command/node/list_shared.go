package node

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-helper/helpers"
)

func computeListRawStruct(nodes []*api.Node, reader helpers.PropReader) ([]map[string]string, error) {
	m := make([]map[string]string, 0)

	for _, node := range nodes {
		val, err := reader.ReadMap(node)
		if err != nil {
			return nil, err
		}

		m = append(m, val)
	}

	return m, nil
}

func listResponse(format string, nodes []*api.Node, propReader helpers.PropReader) (string, error) {
	// Create output based on requested format
	switch format {
	case "table":
		res, err := computeListTableStruct(nodes, propReader)
		if err != nil {
			return "", err
		}

		var b bytes.Buffer
		writer := bufio.NewWriter(&b)
		printListTable(res, propReader, writer)
		writer.Flush()

		return b.String(), nil

	case "json":
		res, err := computeListRawStruct(nodes, propReader)
		if err != nil {
			return "", err
		}

		jsonText, err := json.Marshal(res)
		if err != nil {
			return "", err
		}

		return string(jsonText), nil

	case "json-pretty":
		res, err := computeListRawStruct(nodes, propReader)
		if err != nil {
			return "", err
		}

		jsonText, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return "", err
		}

		return string(jsonText), err

	default:
		return "", fmt.Errorf("Invalid output-format: %s", format)
	}
}
