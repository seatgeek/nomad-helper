package node

import (
	"context"
	"fmt"
	"github.com/hashicorp/nomad/api"
	nomadStructs "github.com/hashicorp/nomad/nomad/structs"
	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"sync"
	"time"
)

func Drain(c *cli.Context, logger *log.Logger) error {
	// Check that enable or disable is not set with monitor
	if c.Bool("monitor") && (c.Bool("enable") || c.Bool("disable")) {
		return fmt.Errorf("-monitor flag cannot be used with the '-enable' or '-disable' flags")
	}

	/*	// Check that we got either enable or disable, but not both.
		if c.Bool("with-benefits") || ((c.Bool("enable") && c.Bool("disable")) || (!c.Bool("monitor") && !c.Bool("enable") && !c.Bool("disable"))) {
			return fmt.Errorf("ethier the '-enable','-disable' or '-with-benefits' flag must be set, unless using '-monitor'")
		}
	*/
	// Validate a compatible set of flags were set
	if c.Bool("disable") && (c.Bool("force") || c.Bool("no-deadline") || c.Bool("ignore-system")) {
		return fmt.Errorf("-disable can't be combined with flags configuring drain strategy")
	}
	if c.Bool("force") && c.Bool("no-deadline") {
		return fmt.Errorf("-force and -no-deadline are mutually exclusive")
	}
	newConstraint := &api.Constraint{}
	if c.Bool("with-benefits") {
		if c.String("constraint") == "" {
			return fmt.Errorf("with-benefits selected, must provide new constrain name")
		}
		if c.String("operand") == "" {
			return fmt.Errorf("with-benefits selected, must provide new constrain name")
		}
		if c.String("value") == "" {
			return fmt.Errorf("with-benefits selected, must provide new constrain name")
		}
		newConstraint = api.NewConstraint(fmt.Sprintf("${%s}", c.String("constraint")), c.String("operand"), c.String("value"))
	}

	deadline := c.Duration("deadline")
	if c.Bool("force") {
		deadline = -1
	}

	if c.Bool("no-deadline") {
		deadline = 0
	}

	// create Nomad API client
	nomadClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	filters := helpers.ClientFilterFromCLI(c.Parent())

	// find nodes to target
	matches, err := helpers.FilteredClientList(nomadClient, false, filters, logger)
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		return fmt.Errorf("could not find any nodes matching provided filters")
	}

	var wg sync.WaitGroup
	ctx := context.Background()

	for _, node := range matches {
		log.Infof("Node %s (class: %s / version: %s)", node.Name, node.NodeClass, node.Attributes["nomad.version"])
		if c.Bool("with-benefits") {
			log.Infof("Drain mode with benefits selected, marking node as ineligible and starting to move the jobs to the specified constraint")
			_, err := nomadClient.Nodes().ToggleEligibility(node.ID, false, nil)
			if err != nil {
				log.Errorf("Error updating scheduling eligibility for %s: %s", node.Name, err)
				continue
			}
			// Bring the allocations running on the node
			nodeAllocations, _, err := nomadClient.Nodes().Allocations(node.ID, nil)
			if err != nil {
				log.Errorf("Error updating scheduling eligibility for %s: %s", node.Name, err)
				continue
			}

			for _, nodeAllocation := range nodeAllocations {
				if *nodeAllocation.Job.Type != nomadStructs.JobTypeService {
					log.Infof("Skipping %s because it's not a service job", nodeAllocation.JobID)
					continue
				}

				if nodeAllocation.ClientStatus == nomadStructs.AllocClientStatusComplete || nodeAllocation.DesiredStatus == nomadStructs.AllocDesiredStatusStop {
					log.Infof("Skipping %s because it's already complete", nodeAllocation.JobID)
					continue
				}

				log.Infof("Found Allocation %s, for job %s, moving it", nodeAllocation.ID, nodeAllocation.JobID)

				allocationJob := nodeAllocation.Job
				existingConstraintAppended := false
				for taskGroupIndex, taskGroup := range allocationJob.TaskGroups {
					if *taskGroup.Name == nodeAllocation.TaskGroup {
						for constraintIndex, constraint := range taskGroup.Constraints {
							if constraint.LTarget == newConstraint.LTarget {
								allocationJob.TaskGroups[taskGroupIndex].Constraints[constraintIndex] = newConstraint
								existingConstraintAppended = true
							}
						}
						if !existingConstraintAppended {
							allocationJob.TaskGroups[taskGroupIndex].Constrain(newConstraint)
						}
					}
				}
				registerResponse, _, err := nomadClient.Jobs().Register(allocationJob, nil)
				if err != nil {
					return fmt.Errorf("failed to move taskgroup %s for job %s: %s", nodeAllocation.TaskGroup, nodeAllocation.JobID, err)
				}
				if c.Bool("wait-for-pending") {
					log.Infof("Waiting for successfully placing the moved job")
					// wait for the evaluation to be available
					for {
						time.Sleep(1 * time.Second)
						evaluation, _, err := nomadClient.Evaluations().Info(registerResponse.EvalID, nil)
						if err != nil {
							continue
						}

						if evaluation.Status == nomadStructs.EvalStatusCancelled || evaluation.Status == nomadStructs.EvalStatusFailed {
							logger.Errorf("Could not evaluate the job: %s", evaluation.StatusDescription)
							continue
						}

						if evaluation.Status == nomadStructs.EvalStatusComplete {
							log.Infof("Evaluation %s for job %s completed", evaluation.ID, nodeAllocation.JobID)
							break
						}
					}

					// wait for the allocations to be available
					for {
						time.Sleep(1 * time.Second)
						evaluations, _, err := nomadClient.Evaluations().List(nil)
						if err != nil {
							continue
						}
						for _, evaluation := range evaluations {
							if &evaluation.JobID == allocationJob.ID && evaluation.Status == nomadStructs.EvalStatusBlocked {
								log.Infof("Job %s got blocked evaluations", allocationJob.ID)
								continue
							}
						}

						break
					}

					// waiting for allocation to be placed
					for {
						pendingAllocations := 0
						allocations, _, err := nomadClient.Allocations().List(nil)
						if err != nil {
							continue
						}
						for _, allocation := range allocations {
							if allocation.JobID == *allocationJob.ID {
								for _, ts := range allocation.TaskStates {
									if ts.State == nomadStructs.TaskStatePending {
										pendingAllocations++
										log.Infof("Allocation %s for job %s is pending,  waiting for this to be resolved", allocation.ID, *allocationJob.ID)
									}
								}
							}
						}
						if pendingAllocations > 0 {
							continue
						}
						break
					}

					log.Infof("All allocations for job %s are not pending anmore", *allocationJob.ID)
				}
				log.Infof("Job %s was successfully moved!", *allocationJob.ID)
			}
			continue
		}

		// in monitor mode we don't do any change to node state
		if c.Bool("monitor") {
			go monitor(ctx, nomadClient, node, &wg)
			continue
		}

		var spec *api.DrainSpec
		if c.Bool("enable") {
			spec = &api.DrainSpec{
				Deadline:         deadline,
				IgnoreSystemJobs: c.Bool("ignore-system"),
			}
		}

		_, err := nomadClient.Nodes().UpdateDrain(node.ID, spec, !c.Bool("keep-ineligible"), nil)
		if err != nil {
			log.Errorf("Could not update drain config for %s: %s", node.Name, err)
			continue
		}

		if !c.Bool("enable") || c.Bool("detach") {
			if c.Bool("enable") {
				log.Infof("Node %q drain strategy set", node.ID)
			} else {
				log.Infof("Node %q drain strategy unset", node.ID)
			}
		}

		if c.Bool("enable") && !c.Bool("detach") {
			go monitor(ctx, nomadClient, node, &wg)
		}
	}

	wg.Wait()

	return nil
}

func monitor(ctx context.Context, client *api.Client, node *api.Node, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	logger := log.WithField("node", node.Name)
	ch := client.Nodes().MonitorDrain(ctx, node.ID, 0, false)
	for {
		select {
		case m, ok := <-ch:
			if !ok { // channel closed
				return
			}

			switch m.Level {
			case api.MonitorMsgLevelNormal:
				logger.Info(m.String())

			case api.MonitorMsgLevelInfo:
				logger.Info(m.String())

			case api.MonitorMsgLevelWarn:
				logger.Warn(m.String())

			case api.MonitorMsgLevelError:
				logger.Error(m.String())
			}

		}
	}
}
