package reevaluate

import (
	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-helper/nomad"
)

func App() error {
	client, err := nomad.NewNomadClient()
	if err != nil {
		return err
	}

	jobStubs, _, err := client.Jobs().List(&api.QueryOptions{})

	for _, jobStub := range jobStubs {
		log.Infof("Evaluating %s", jobStub.Name)
		_, _, err := client.Jobs().ForceEvaluate(jobStub.ID, &api.WriteOptions{})
		if err != nil {
			log.Errorf("  %s", err)
		}
	}

	return nil
}
