package helpers

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Jeffail/tunny"
	"github.com/hashicorp/nomad/api"
	"github.com/karlseguin/ccache"
	"github.com/schollz/progressbar/v2"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	nodeCache = ccache.New(ccache.Configure().MaxSize(5000).ItemsToPrune(10))
	stderrLog *log.Logger
)

type ClientFilter struct {
	Attribute   []string
	Class       string
	Eligibility string
	Meta        []string
	NOOP        bool
	Percent     int
	Prefix      string
	Version     string
}

func ClientFilterFromCLI(c *cli.Context) ClientFilter {
	filter := ClientFilter{
		Attribute:   DeleteEmpty(c.StringSlice("filter-attribute")),
		Class:       c.String("filter-class"),
		Eligibility: c.String("filter-eligibility"),
		Meta:        DeleteEmpty(c.StringSlice("filter-meta")),
		NOOP:        c.Bool("noop"),
		Percent:     c.Int("percent"),
		Prefix:      c.String("filter-prefix"),
		Version:     c.String("filter-version"),
	}

	return filter
}

func ClientFilterFromWeb(r *http.Request) ClientFilter {
	filter := ClientFilter{
		Attribute:   DeleteEmpty(strings.Split(r.URL.Query().Get("filter-attribute"), ",")),
		Class:       r.URL.Query().Get("filter-class"),
		Eligibility: r.URL.Query().Get("filter-eligibility"),
		Meta:        DeleteEmpty(strings.Split(r.URL.Query().Get("filter-meta"), ",")),
		Percent:     100,
		Prefix:      r.URL.Query().Get("filter-prefix"),
		Version:     r.URL.Query().Get("filter-version"),
	}

	return filter
}

func init() {
	// We make an logger that _always_ print to stderr to ensure CLI calls like
	// "nomad-helper stats --output-format json | jq '.'" works. All progress and filtering
	// output will go to stderr now instead of stdout for debug/info etc
	stderrLog = log.New()
	stderrLog.Out = os.Stderr
}

func FilteredClientList(client *api.Client, progress bool, filter ClientFilter, logger *log.Logger) ([]*api.Node, error) {
	stderrLog.SetLevel(logger.GetLevel())

	stderrLog.Info("Finding eligible nodes")
	nodes, _, err := client.Nodes().List(&api.QueryOptions{Prefix: filter.Prefix})
	if err != nil {
		return nil, err
	}

	// Configure progressbar
	var bar *progressbar.ProgressBar
	if progress {
		bar = progressbar.NewOptions(len(nodes), progressbar.OptionSetWriter(os.Stderr))
	}

	// Configure worker pool
	pool := tunny.NewFunc(runtime.NumCPU()*2, readNodeWorker(filter, client))
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

			if bar != nil {
				defer bar.Add(1)
			}

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
	if progress {
		bar.Finish()
		fmt.Fprintln(os.Stderr, "")
	}

	stderrLog.Infof("Found %d matched nodes", len(matches))

	// only work on specific percent of nodes
	if percent := filter.Percent; percent > 0 && percent < 100 {
		stderrLog.Infof("Only %d percent of nodes should be used", percent)
		matches = matches[0 : len(matches)*percent/100]
	}

	// noop mode will fail the matching to prevent any further processing
	if filter.NOOP {
		for _, node := range matches {
			stderrLog.Infof("Node %s matched!", node.Name)
		}
		return nil, fmt.Errorf("noop mode, aborting")
	}

	return matches, nil
}

func readNodeWorker(filter ClientFilter, client *api.Client) func(payload interface{}) interface{} {
	return func(payload interface{}) interface{} {
		nodeStub := payload.(*api.NodeListStub)

		// only consider nodes that is ready
		if nodeStub.Status != "ready" {
			stderrLog.Debugf("Node %s is not in status=ready (%s)", nodeStub.Name, nodeStub.Status)
			return nil
		}

		// only consider nodes with the right node class
		if class := filter.Class; class != "" && nodeStub.NodeClass != class {
			stderrLog.Debugf("Node %s class '%s' do not match expected value '%s'", nodeStub.Name, nodeStub.NodeClass, class)
			return nil
		}

		// only consider nodes with the right nomad version
		if version := filter.Version; version != "" && nodeStub.Version != version {
			stderrLog.Debugf("Node %s version '%s' do not match expected node version '%s'", nodeStub.Name, nodeStub.Version, version)
			return nil
		}

		// only consider nodes with the right eligibility
		if eligibility := filter.Eligibility; eligibility != "" && nodeStub.SchedulingEligibility != eligibility {
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
		if meta := filter.Meta; len(meta) > 0 {
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
		if meta := filter.Attribute; len(meta) > 0 {
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
	item, err := nodeCache.Fetch(nodeID, 1*time.Minute, func() (interface{}, error) {
		node, _, err := client.Nodes().Info(nodeID, nil)
		if err != nil {
			return nil, err
		}

		return node, nil
	})

	if err != nil {
		return nil, err
	}

	return item.Value().(*api.Node), nil
}
