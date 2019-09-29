package node

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
)

func BreakdownWeb(logger *log.Logger, r *http.Request) (string, error) {
	// Get list of CLI arguments we should use as dimensions
	dimensions := helpers.DeleteEmpty(strings.Split(r.URL.Path, "/"))
	if len(dimensions) == 0 {
		return "", fmt.Errorf("Missing path (see help docs)")
	}

	// Create filters
	filters := helpers.ClientFilterFromWeb(r)

	// Collect Node data from the Nomad cluster
	nodes, err := getData(filters, logger, false)
	if err != nil {
		return "", err
	}

	// Create a prop reader for results
	propReader := helpers.NewMetaPropReader(dimensions...)

	// Output result
	return breakdownResponse(r.URL.Query().Get("output-format"), nodes, propReader)
}
