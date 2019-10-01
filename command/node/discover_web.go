package node

import (
	"net/http"

	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
)

func DiscoverWeb(logger *log.Logger, r *http.Request) (string, error) {
	// Create filters
	filters := helpers.ClientFilterFromWeb(r)

	// Collect Node data from the Nomad cluster
	result, err := discoverData(filters, logger, false)
	if err != nil {
		return "", err
	}

	// Decide on output format
	format := r.URL.Query().Get("output-format")
	if format == "" {
		format = "table"
	}

	// Output result
	return discoverResponse(format, *result)
}
