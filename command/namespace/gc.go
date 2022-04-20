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

	var deletableNamespaces []*api.Namespace
	namespaces, _, err := nomadClient.Namespaces().List(nil)
	for _, namespace := range namespaces {

		jobs, _, err := nomadClient.Jobs().List(&api.QueryOptions{Namespace: namespace.Name})
		if err != nil {
			return err
		}

		if len(jobs) == 0 {
			deletableNamespaces = append(deletableNamespaces, namespace)
		} else {
			log.Infof("Cannot delete namespace %s, %d jobs in namespace", namespace.Name, len(jobs))
		}
	}

	for _, namespace := range deletableNamespaces {
		if c.Bool("dry") {
			logger.Infof("Skipping deletion of namespace %s because dry flag was provided", namespace.Name)
			continue
		}

		if _, err = nomadClient.Namespaces().Delete(namespace.Name, nil); err != nil {
			return fmt.Errorf("error deleting namespace: %s", err.Error())
		}

		log.Infof("Namespace %s was successfully deleted!", namespace.Name)
	}

	return nil
}
