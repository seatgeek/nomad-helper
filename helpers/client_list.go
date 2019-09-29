package helpers

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/Jeffail/tunny"
	"github.com/hashicorp/nomad/api"
	"github.com/schollz/progressbar/v2"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	nodeCache = make(map[string]*api.Node)
	stderrLog *log.Logger
	cacheLock sync.RWMutex
)

func init() {
	// We make an logger that _always_ print to stderr to ensure CLI calls like
	// "nomad-helper stats --output-format json | jq '.'" works. All progress and filtering
	// output will go to stderr now instead of stdout for debug/info etc
	stderrLog = log.New()
	stderrLog.Out = os.Stderr
}

func FilteredClientList(client *api.Client, c *cli.Context, logger *log.Logger) ([]*api.Node, error) {
	stderrLog.SetLevel(logger.GetLevel())

	stderrLog.Info("Finding eligible nodes")
	nodes, _, err := client.Nodes().List(&api.QueryOptions{Prefix: c.String("filter-prefix")})
	if err != nil {
		return nil, err
	}

	// Configure progressbar
	bar := progressbar.NewOptions(len(nodes), progressbar.OptionSetWriter(os.Stderr))
	if !c.Bool("no-progress") {
		bar.RenderBlank()
		defer func() {
			bar.Finish()
		}()
	}

	// Configure worker pool
	pool := tunny.NewFunc(runtime.NumCPU()*2, readNodeWorker(c, bar, client))
	defer pool.Close()

	// Lucks & wait groups
	var wg sync.WaitGroup
	var l sync.Mutex

	matches := make([]*api.Node, 0)

	// Iterate all matched nodes
	for _, client := range nodes {
		wg.Add(1)

		// Spin up a go-routine for fetching filtering and reading out the node details
		go func(c *api.NodeListStub) {
			defer wg.Done()

			r := pool.Process(c)
			if r == nil {
				return
			}

			l.Lock()
			matches = append(matches, r.(*api.Node))
			l.Unlock()
		}(client)
	}

	// Wait for all workers to complete
	wg.Wait()

	// Complete progressbar if needed
	if !c.Bool("no-progress") {
		bar.Finish()
		fmt.Fprintln(os.Stderr, "")
	}

	stderrLog.Infof("Found %d matched nodes", len(matches))

	// only work on specific percent of nodes
	if percent := c.Int("percent"); percent > 0 && percent < 100 {
		stderrLog.Infof("Only %d percent of nodes should be used", percent)
		matches = matches[0 : len(matches)*percent/100]
	}

	// noop mode will fail the matching to prevent any further processing
	if c.BoolT("noop") {
		for _, node := range matches {
			stderrLog.Infof("Node %s matched!", node.Name)
		}
		return nil, fmt.Errorf("noop mode, aborting")
	}

	return matches, nil
}

func readNodeWorker(c *cli.Context, bar *progressbar.ProgressBar, client *api.Client) func(payload interface{}) interface{} {
	return func(payload interface{}) interface{} {
		nodeStub := payload.(*api.NodeListStub)

		defer func() {
			if !c.Bool("no-progress") {
				bar.Add(1)
			}
		}()

		// only consider nodes that is ready
		if nodeStub.Status != "ready" {
			stderrLog.Debugf("Node %s is not in status=ready (%s)", nodeStub.Name, nodeStub.Status)
			return nil
		}

		// only consider nodes with the right node class
		if class := c.String("filter-class"); class != "" && nodeStub.NodeClass != class {
			stderrLog.Debugf("Node %s class '%s' do not match expected value '%s'", nodeStub.Name, nodeStub.NodeClass, class)
			return nil
		}

		// only consider nodes with the right nomad version
		if version := c.String("filter-version"); version != "" && nodeStub.Version != version {
			stderrLog.Debugf("Node %s version '%s' do not match expected node version '%s'", nodeStub.Name, nodeStub.Version, version)
			return nil
		}

		// only consider nodes with the right eligibility
		if eligibility := c.String("filter-eligibility"); eligibility != "" && nodeStub.SchedulingEligibility != eligibility {
			stderrLog.Debugf("Node %s eligibility '%s' do not match expected node eligibility '%s'", nodeStub.Name, nodeStub.SchedulingEligibility, eligibility)
			return nil
		}

		// Read full Node info from Nomad
		node, err := lookupNode(nodeStub.ID, client)
		if err != nil {
			stderrLog.Error(err)
			return nil
		}

		// filter by client meta keys
		if meta := c.StringSlice("filter-meta"); len(meta) > 0 {
			for _, chunk := range meta {
				split := strings.Split(chunk, "=")
				if len(split) != 2 {
					stderrLog.Fatalf("Could not marge filter-meta '%s' as 'key=value' pair", chunk)
					return nil
				}

				key := split[0]
				value := split[1]

				if nodeValue := getNodeMetaProperty(node, key); nodeValue != value {
					stderrLog.Debugf("Node %s Meta key '%s' value '%s' do not match expected '%s'", nodeStub.Name, key, nodeValue, value)
					return nil
				}
			}
		}

		// filter by client attribute keys
		if meta := c.StringSlice("filter-attribute"); len(meta) > 0 {
			for _, chunk := range meta {
				split := strings.Split(chunk, "=")
				if len(split) != 2 {
					stderrLog.Fatalf("Could not marge filter-meta '%s' as 'key=value' pair", chunk)
					return nil
				}

				key := split[0]
				value := split[1]

				if nodeValue := getNodeAttributesProperty(node, key); nodeValue != value {
					stderrLog.Debugf("Node %s Attribute key '%s' value '%s' do not match expected '%s'", nodeStub.Name, key, nodeValue, value)
					return nil
				}
			}
		}

		// continue to furhter processing
		stderrLog.Debugf("Node %s passed all filters", nodeStub.Name)
		return node
	}
}

func getNodeMetaProperty(node *api.Node, key string) string {
	d, ok := node.Meta[key]
	if !ok {
		return "__not_found__"
	}
	return d
}

func getNodeAttributesProperty(node *api.Node, key string) string {
	d, ok := node.Attributes[key]
	if !ok {
		return "__not_found__"
	}
	return d
}

func lookupNode(nodeID string, client *api.Client) (*api.Node, error) {
	cacheLock.RLock()
	data, ok := nodeCache[nodeID]
	cacheLock.RUnlock()

	if !ok {
		node, _, err := client.Nodes().Info(nodeID, nil)
		if err != nil {
			return nil, err
		}

		cacheLock.Lock()
		nodeCache[nodeID] = node
		cacheLock.Unlock()

		return node, nil
	}

	return data, nil
}
