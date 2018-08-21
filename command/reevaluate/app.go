package reevaluate

import (
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-helper/nomad"
	log "github.com/sirupsen/logrus"
)

func App() error {
	client, err := nomad.NewNomadClient()
	if err != nil {
		return err
	}

	jobStubs, _, err := client.Jobs().List(&api.QueryOptions{})

	for _, jobStub := range jobStubs {
		if strings.Contains(jobStub.ID, "/periodic-") {
			log.Info("Skipping %s - periodic job", jobStub.Name)
			continue
		}

		log.Infof("Evaluating %s", jobStub.Name)
		_, _, err := client.Jobs().EvaluateWithOpts(jobStub.ID, api.EvalOptions{ForceReschedule: true}, &api.WriteOptions{})
		if err != nil {
			log.Errorf("  %s", err)
		}
	}

	return nil
}
