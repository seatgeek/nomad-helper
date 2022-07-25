package namespace

import (
	"fmt"

	"github.com/hashicorp/nomad/api"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

func GC(c *cli.Context, logger *log.Logger) error {
	nomadClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	regions, err := nomadClient.Regions().List()
	if err != nil {
		return err
	}

	var deletableNamespaces []*api.Namespace
	namespaces, _, err := nomadClient.Namespaces().List(nil)
	for _, namespace := range namespaces {
		var jobs []*api.JobListStub
		for _, region := range regions {
			regionJobs, _, err := nomadClient.Jobs().List(&api.QueryOptions{
				Region:    region,
				Namespace: namespace.Name,
			})
			if err != nil {
				return err
			}

			for _, job := range regionJobs {
				if inStringSlice(job.ID, c.StringSlice("ignore-job")) {
					log.Infof("Ignoring job '%s' in region/namespace '%s/%s'", job.ID, region, namespace.Name)
					continue
				}
				jobs = append(jobs, job)
			}
		}

		if len(jobs) == 0 {
			deletableNamespaces = append(deletableNamespaces, namespace)
		} else {
			log.Infof("Cannot delete namespace %s, %d jobs in namespace", namespace.Name, len(jobs))
		}
	}

	for _, namespace := range deletableNamespaces {
		for _, region := range regions {
			jobs, _, err := nomadClient.Jobs().List(&api.QueryOptions{
				Region:    region,
				Namespace: namespace.Name,
			})
			if err != nil {
				return err
			}

			if len(jobs) == 0 {
				continue
			}

			if c.Bool("dry") {
				logger.Infof("Skipping deletion of %d jobs in region/namespace %s/%s because dry flag was provided", len(jobs), region, namespace.Name)
				continue
			}

			for _, job := range jobs {
				// Ideally we also track the evalID state but we'd need to duplicate
				// all the monitor logic from the nomad codebase as it's not exposed
				_, _, err := nomadClient.Jobs().Deregister(job.ID, true, &api.WriteOptions{
					Region:    region,
					Namespace: namespace.Name,
				})
				if err != nil {
					return fmt.Errorf("error deleting job '%s' in region/namespace '%s/%s': %w", job.ID, region, namespace.Name, err)
				}
				logger.Infof("Job '%s' in region/namespace '%s/%s' successfully deleted", job.ID, region, namespace.Name)
			}
		}
	}

	if !c.Bool("dry") {
		logger.Infof("executing garbage collection")
		if err := nomadClient.System().GarbageCollect(); err != nil {
			return fmt.Errorf("error running garbage collection: %w", err)
		}

		logger.Infof("executing summary reconciliation")
		if err := nomadClient.System().ReconcileSummaries(); err != nil {
			return fmt.Errorf("error reconciling summaries: %w", err)
		}
	}

	for _, namespace := range deletableNamespaces {
		if c.Bool("dry") {
			logger.Infof("Skipping deletion of namespace %s because dry flag was provided", namespace.Name)
			continue
		}

		if _, err = nomadClient.Namespaces().Delete(namespace.Name, nil); err != nil {
			return fmt.Errorf("error deleting namespace: %w", err)
		}

		log.Infof("Namespace %s was successfully deleted!", namespace.Name)
	}

	return nil
}

func inStringSlice(s string, ss []string) bool {
	if len(ss) > 0 {
		for _, item := range ss {
			if s == item {
				return true
			}
		}
	}

	return false
}
