package job

import (
	"fmt"
	"sync"

	"github.com/hashicorp/nomad/api"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func Stop(c *cli.Context, logger *log.Logger) error {
	jobName := c.Args().First()
	if jobName == "" {
		return fmt.Errorf("Must provide a job name or prefix")
	}

	// create Nomad API client
	nomadClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	jobsToStop := []string{jobName}

	// if we stop by prefix, then query for all the jobs
	if c.Bool("as-prefix") {
		jobsToStop = []string{}

		jobs, _, err := nomadClient.Jobs().List(&api.QueryOptions{Prefix: jobName})
		if err != nil {
			return err
		}

		for _, job := range jobs {
			jobsToStop = append(jobsToStop, job.ID)
		}
	}

	if len(jobsToStop) == 0 {
		return fmt.Errorf("Could not find any jobs")
	}

	var wg sync.WaitGroup
	for _, jobName := range jobsToStop {
		logger.Infof("Going to stop job %s", jobName)

		if c.Bool("dry") {
			continue
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			_, _, err := nomadClient.Jobs().Deregister(name, c.Bool("purge"), nil)
			if err != nil {
				log.Error(err)
				return
			}

			log.Infof("Job %s was successfully stopped!", name)
		}(jobName)
	}

	wg.Wait()

	return nil
}
