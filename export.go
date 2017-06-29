package main

import (
	"io/ioutil"
	"time"

	yaml "gopkg.in/yaml.v2"

	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/nomad/api"
)

func exportCommand(file string) error {
	log.Info("Reading jobs from Nomad")

	client, err := NewNomadClient()
	if err != nil {
		return err
	}

	jobStubs, _, err := client.Jobs().List(&api.QueryOptions{})

	if err != nil {
		return err
	}

	info := make(map[string]string)
	info["nomad_addr"] = os.Getenv("NOMAD_ADDR")
	info["exported_at"] = time.Now().UTC().Format(time.RFC1123Z)
	info["exported_by"] = os.Getenv("USER")

	state := &NomadState{
		Info: info,
		Jobs: make(map[string]TaskGroupState),
	}

	for _, jobStub := range jobStubs {
		log.Debugf("Scanning job %s", jobStub.Name)

		job, _, err := client.Jobs().Info(jobStub.Name, &api.QueryOptions{})
		if err != nil {
			log.Errorf("Could not fetch job %s", jobStub.Name)
		}

		jobState := TaskGroupState{}

		for _, group := range job.TaskGroups {
			log.Infof("%s -> %s = %d", jobStub.Name, *group.Name, *group.Count)

			jobState[*group.Name] = *group.Count
		}

		state.Jobs[*job.ID] = jobState
	}

	bytes, err := yaml.Marshal(state)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(file, bytes, 0644)
	if err != nil {
		return err
	}

	log.Info("Nomad state was successfully written out")

	return nil
}
