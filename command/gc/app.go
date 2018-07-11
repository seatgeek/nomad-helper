package gc

import (
	"github.com/seatgeek/nomad-helper/nomad"
	log "github.com/sirupsen/logrus"
)

func App() error {
	client, err := nomad.NewNomadClient()
	if err != nil {
		return err
	}

	err = client.System().GarbageCollect()
	if err != nil {
		return err
	}

	log.Info("Done!")
	return nil
}
