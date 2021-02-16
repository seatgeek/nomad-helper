package job

import (
	"fmt"
	nomad "github.com/hashicorp/nomad/api"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"strings"
	"sync"
)

func Move(c *cli.Context, logger *log.Logger) error {
	jobName := c.Args().First()

	// Sanity check
	if jobName == "" {
		return fmt.Errorf("must provide a job name or prefix")
	}
	if c.String("constraint") == "" {
		return fmt.Errorf("must provide new constrain name")
	}
	if c.String("operand") == "" {
		return fmt.Errorf("must provide new constrain name")
	}
	if c.String("value") == "" {
		return fmt.Errorf("must provide new constrain name")
	}

	newConstraint := nomad.NewConstraint(fmt.Sprintf("${%s}", c.String("constraint")), c.String("operand"), c.String("value"))

	// create Nomad API client
	nomadClient, err := nomad.NewClient(nomad.DefaultConfig())
	if err != nil {
		return err
	}

	jobsToMove := []string{jobName}

	// if we stop by prefix, then query for all the jobs
	if c.Bool("as-prefix") {
		jobsToMove = []string{}

		jobs, _, err := nomadClient.Jobs().List(&nomad.QueryOptions{Prefix: jobName})
		if err != nil {
			return err
		}

		for _, job := range jobs {
			if excludeFilter := c.String("exclude"); excludeFilter != "" {
				if strings.Contains(job.ID, excludeFilter) {
					continue
				}
			}
			jobsToMove = append(jobsToMove, job.ID)
		}
	}

	if len(jobsToMove) == 0 {
		return fmt.Errorf("could not find any jobs")
	}

	var wg sync.WaitGroup
	for _, jobName := range jobsToMove {
		logger.Infof("Going to move job %s", jobName)

		if c.Bool("dry") {
			logger.Infof("Skipping, because <dry> flag was provided")
			continue
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			job, _, err := nomadClient.Jobs().Info(name, nil)
			if err != nil {
				log.Error(err)
				return
			}
			log.Debugf("Read the job %v", job)
			existingConstraintAppended := false
			for _, constraint := range job.Constraints {
				if constraint.LTarget == newConstraint.LTarget {
					constraint = newConstraint
					existingConstraintAppended = true
				}
			}
			if !existingConstraintAppended {
				job.Constraints = append(job.Constraints, newConstraint)
			}
			response, _, err := nomadClient.Jobs().Register(job, nil)
			if err != nil {
				log.Error(err)
				return
			}
			log.Debug(response)
			log.Infof("Job %s was successfully moved!", name)
		}(jobName)
	}

	wg.Wait()

	return nil
}
