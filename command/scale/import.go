package scale

import (
	"io/ioutil"

	"github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-helper/nomad"
	"github.com/seatgeek/nomad-helper/structs"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

func ImportCommand(file string) error {
	log.Info("Reading state file")

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	localState := &structs.NomadState{}
	err = yaml.Unmarshal(data, &localState)
	if err != nil {
		return err
	}

	client, err := nomad.NewNomadClient()
	if err != nil {
		return err
	}

	for localJobName, jobGroups := range localState.Jobs {
		logger := log.WithField("job", localJobName)

		remoteJob, _, err := client.Jobs().Info(localJobName, &api.QueryOptions{})
		if err != nil {
			logger.Errorf("Could not find remote job: %s", err)
			continue
		}

		// Test if we can find the local group state group name in the remote job
		foundRemoteGroup := false
		shouldUpdate := false
		oldCount := 0

		for localGroupName, localGroupCount := range jobGroups {
			for i, jobGroup := range remoteJob.TaskGroups {
				// Name doesn't match
				if localGroupName != *jobGroup.Name {
					continue
				}

				foundRemoteGroup = true

				// Don't bother to update if the count is already the same
				if *jobGroup.Count == localGroupCount {
					logger.Info("Skipping update since remote and local count is the same")
					break
				}

				// Update the remote count
				oldCount = *jobGroup.Count

				remoteJob.TaskGroups[i].Count = &localGroupCount

				logger.Infof("Will change group count from %d to %d", oldCount, localGroupCount)

				shouldUpdate = true
				break
			}

			// If we could not find the job, alert and move on to the next
			if !foundRemoteGroup {
				logger.Error("Could not find the group in remote cluster job")
				continue
			}
		}

		if shouldUpdate {
			_, _, err = client.Jobs().Register(remoteJob, &api.WriteOptions{})
			if err != nil {
				log.Error(err)
				continue
			}
		}
	}

	return nil
}
