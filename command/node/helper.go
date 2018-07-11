package node

import (
	"fmt"

	"github.com/hashicorp/nomad/api"
	log "github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
)

var (
	nodeCache = make(map[string]*api.Node)
)

func filter(client *api.Client, c *cli.Context) ([]*api.NodeListStub, error) {
	nodes, _, err := client.Nodes().List(&api.QueryOptions{Prefix: c.String("filter-prefix")})
	if err != nil {
		return nil, err
	}

	matches := make([]*api.NodeListStub, 0)
	for _, node := range nodes {
		// only consider nodes that is ready
		if node.Status != "ready" {
			log.Debugf("Node %s (%s) is not in status=ready (%s)", node.Name, node.NodeClass, node.Status)
			continue
		}

		// only consider nodes with the right node class
		if class := c.String("filter-class"); class != "" && node.NodeClass != class {
			log.Debugf("Node %s (%s) to not match node class %s", node.Name, node.NodeClass, class)
			continue
		}

		// only consider nodes with the right nomad version
		if version := c.String("filter-nomad-version"); version != "" && node.Version != version {
			log.Debugf("Node %s (%s) to not match node version %s", node.Name, node.Version, version)
			continue
		}

		// only consider nodes with the right base ami version
		if version := c.String("filter-ami-version"); version != "" {
			if amiVersion := getNodeMetaProperty(node.ID, "aws.ami-version", client); amiVersion != version {
				log.Debugf("Node %s (%s) AMI version do not match %s", node.Name, version, amiVersion)
				continue
			}
		}

		// continue to furhter processing
		log.Debugf("Node %s (%s) is still good", node.Name, node.NodeClass)
		matches = append(matches, node)

		// noop mode should just print the nodes right away
		if c.Bool("noop") {
			log.Infof("Node %s (class: %s / version: %s)", node.Name, node.NodeClass, node.Version)
		}
	}

	log.Infof("Found %d matched nodes", len(matches))

	// noop mode will fail the matching to prevent any further processing
	if c.Bool("noop") {
		return nil, fmt.Errorf("noop mode, aborting")
	}

	return matches, nil
}

func hasFilter(c *cli.Context, field string) bool {
	return c.String(field) != ""
}

func getNodeMetaProperty(nodeID string, key string, client *api.Client) string {
	node, err := lookupNode(nodeID, client)
	if err != nil {
		log.Errorf("Could not lookup the node in Nomad API: %s", err)
		return ""
	}

	// spew.Dump(node)
	d, ok := node.Meta[key]
	if !ok {
		return ""
	}
	return d
}

func lookupNode(nodeID string, client *api.Client) (*api.Node, error) {
	data, ok := nodeCache[nodeID]
	if !ok {
		node, _, err := client.Nodes().Info(nodeID, nil)
		if err != nil {
			return nil, err
		}

		nodeCache[nodeID] = node
		return node, nil
	}

	return data, nil

}
