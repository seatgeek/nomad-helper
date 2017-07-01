package main

import (
	log "github.com/Sirupsen/logrus"
)

func gcCommand() error {
	client, err := NewNomadClient()
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
