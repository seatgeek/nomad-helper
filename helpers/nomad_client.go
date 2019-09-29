package helpers

import (
	"fmt"
	"strings"

	api "github.com/hashicorp/nomad/api"
)

type PropReader interface {
	Read(node *api.Node) ([]string, error)
	ReadMap(node *api.Node) (map[string]string, error)
	GetKeys() []string
}

func NewMetaPropReader(props ...string) PropReader {
	return Reader{keys: props}
}

type Reader struct {
	keys []string
}

func (r Reader) GetKeys() []string {
	return r.keys
}

func (r Reader) Read(node *api.Node) ([]string, error) {
	s := make([]string, 0)

	for _, prop := range r.keys {
		val, err := r.getPropValue(prop, node)
		if err != nil {
			return nil, err
		}

		s = append(s, val)
	}

	return s, nil
}

func (r Reader) ReadMap(node *api.Node) (map[string]string, error) {
	s := make(map[string]string, 0)

	for _, prop := range r.keys {
		val, err := r.getPropValue(prop, node)
		if err != nil {
			return nil, err
		}

		s[prop] = val
	}

	return s, nil
}

func (r *Reader) getPropValue(prop string, node *api.Node) (string, error) {
	chunks := strings.Split(prop, ".")

	// we lower key 0 to make matching simpler
	switch strings.ToLower(chunks[0]) {
	case "attributes", "attribute":
		key := strings.Join(chunks[1:], ".")
		value, ok := node.Attributes[key]
		if !ok {
			return "- missing -", nil
		}
		return value, nil

	// Common attribute shortcuts
	case "hostname":
		return node.Attributes["unique.hostname"], nil

	case "ip", "address", "ip-address":
		return node.Attributes["unique.network.ip-address"], nil

	case "meta":
		key := strings.Join(chunks[1:], ".")
		value, ok := node.Meta[key]
		if !ok {
			return "- missing -", nil
		}
		return value, nil

	case "nodeclass", "class":
		return node.NodeClass, nil

	case "id":
		return node.ID, nil

	case "name":
		return node.Name, nil

	case "datacenter", "dc":
		return node.Datacenter, nil

	case "drain":
		return fmt.Sprintf("%+v", node.Drain), nil

	case "status":
		return node.Status, nil

	case "schedulingeligibility", "eligibility":
		return node.SchedulingEligibility, nil

	default:
		return "", fmt.Errorf("Don't know how to find value for '%s'", prop)
	}
}
