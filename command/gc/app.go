package gc

import (
	log "github.com/Sirupsen/logrus"
	"github.com/seatgeek/nomad-helper/nomad"
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
