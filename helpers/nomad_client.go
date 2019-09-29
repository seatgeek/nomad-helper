package helpers

import (
	"fmt"
	"strings"

	api "github.com/hashicorp/nomad/api"
)

type PropReader interface {
	Read(node *api.Node) []string
	ReadMap(node *api.Node) map[string]string
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

func (r Reader) Read(node *api.Node) []string {
	s := make([]string, 0)

	for _, prop := range r.keys {
		s = append(s, r.getPropValue(prop, node))
	}

	return s
}

func (r Reader) ReadMap(node *api.Node) map[string]string {
	s := make(map[string]string, 0)

	for _, prop := range r.keys {
		s[prop] = r.getPropValue(prop, node)
	}

	return s
}

func (r *Reader) getPropValue(prop string, node *api.Node) string {
	chunks := strings.Split(prop, ".")

	// we lower key 0 to make matching simpler
	switch strings.ToLower(chunks[0]) {
	case "attributes", "attribute":
		key := strings.Join(chunks[1:], ".")
		value, ok := node.Attributes[key]
		if !ok {
			return "- missing -"
		}
		return value

	// Common attribute shortcuts
	case "hostname":
		return node.Attributes["unique.hostname"]

	case "ip", "address", "ip-address":
		return node.Attributes["unique.network.ip-address"]

	case "meta":
		key := strings.Join(chunks[1:], ".")
		value, ok := node.Meta[key]
		if !ok {
			return "- missing -"
		}
		return value

	case "nodeclass", "class":
		return node.NodeClass

	case "id":
		return node.ID

	case "name":
		return node.Name

	case "datacenter", "dc":
		return node.Datacenter

	case "drain":
		return fmt.Sprintf("%+v", node.Drain)

	case "status":
		return node.Status

	case "schedulingeligibility", "eligibility":
		return node.SchedulingEligibility

	default:
		panic(fmt.Sprintf("don't know how to find value for '%s'", prop))
	}
}
