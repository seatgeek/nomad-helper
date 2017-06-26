package main

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/nomad/api"
)

func importCommand(file string) error {
	log.Info("Reading state file")

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	localState := &NomadState{}
	err = yaml.Unmarshal(data, &localState)
	if err != nil {
		return err
	}

	client, err := NewNomadClient()
	if err != nil {
		return err
	}

	modifiedJobs := make(map[string]*api.Job)

	for _, localGroupState := range localState.Groups {
		log.Infof("Processing %s -> %s", localGroupState.Job, localGroupState.Group)

		index := localGroupState.Job + "_" + localGroupState.Group

		// see if we have processed the job already
		remoteJob, existingJob := modifiedJobs[index]
		if !existingJob {
			remoteJob, _, err = client.Jobs().Info(localGroupState.Job, &api.QueryOptions{})
			if err != nil {
				log.Errorf("  Could not find remote job: %s", err)
				continue
			}
		}

		// Test if we can find the local group state group name in the remote job
		foundRemoteGroup := false
		shouldUpdate := true
		oldCount := 0

		for i, jobGroup := range remoteJob.TaskGroups {
			if *jobGroup.Name != localGroupState.Group {
				continue
			}

			foundRemoteGroup = true

			// Don't bother to update if the count is already the same
			if *jobGroup.Count == localGroupState.Count {
				shouldUpdate = false
				log.Info("  Skipping update since remote and local count is the same")
				break
			}

			// Update the remote count
			oldCount = int(*jobGroup.Count)

			remoteJob.TaskGroups[i].Count = &localGroupState.Count
			break
		}

		// If we could not find the job, alert and move on to the next
		if !foundRemoteGroup {
			log.Error("  Could not find the group in remote cluster job")
			continue
		}

		//
		if !shouldUpdate {
			continue
		}

		log.Infof("  Will change group count from %d to %d", oldCount, localGroupState.Count)

		// Add or overwrite the remote job reference in modified jobs
		modifiedJobs[index] = remoteJob

		_, _, err := client.Jobs().Register(remoteJob, &api.WriteOptions{})
		if err != nil {
			log.Error(err)
			continue
		}

		log.Infof("  Successfully updated job %s", *remoteJob.ID)
	}

	return nil
}
