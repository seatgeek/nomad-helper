package main

import (
	"os"
	"sort"

	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/seatgeek/nomad-helper/command/drain"
	"github.com/seatgeek/nomad-helper/command/firehose"
	"github.com/seatgeek/nomad-helper/command/gc"
	"github.com/seatgeek/nomad-helper/command/reevaluate"
	"github.com/seatgeek/nomad-helper/command/scale"
	cli "gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "nomad-scale-helper"
	app.Usage = "easily restore / snapshot your nomad job scale config"
	app.Version = "0.1"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "log-level",
			Value:  "info",
			Usage:  "Debug level (debug, info, warn/warning, error, fatal, panic)",
			EnvVar: "LOG_LEVEL",
		},
	}
	app.Commands = []cli.Command{
{
			Name:  "node",
			Usage: "node specific commands that act on all Nomad clients that match the filters provided, rather than a single node",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "filter-prefix",
					Usage: "Filter nodes by their ID (prefix matching)",
				},
				cli.StringFlag{
					Name:  "filter-class",
					Usage: "Filter nodes by their node class",
				},
				cli.StringFlag{
					Name:  "filter-nomad-version",
					Usage: "Filter nodes by their Nomad version",
				},
				cli.StringFlag{
					Name:  "filter-ami-version",
					Usage: "Filter nodes by their Instance AMI version (BaseAMI)",
				},
				cli.BoolFlag{
					Name:  "noop",
					Usage: "Only output nodes that would be drained, don't do any modifications",
				},
			},
			Subcommands: []cli.Command{
				{
					Name:  "drain",
					Usage: "The node drain command is used to toggle drain mode on a given node. Drain mode prevents any new tasks from being allocated to the node, and begins migrating all existing allocations away",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "enable",
							Usage: "Enable node drain mode",
						},
						cli.BoolFlag{
							Name:  "disable",
							Usage: "Disable node drain mode",
						},
						cli.DurationFlag{
							Name:  "deadline",
							Usage: "Set the deadline by which all allocations must be moved off the node. Remaining allocations after the deadline are force removed from the node. Defaults to 1 hour",
							Value: time.Hour,
						},
						cli.BoolFlag{
							Name:  "no-deadline",
							Usage: "No deadline allows the allocations to drain off the node without being force stopped after a certain deadline",
						},
						cli.BoolFlag{
							Name:  "monitor",
							Usage: "Enter monitor mode directly without modifying the drain status",
						},
						cli.BoolFlag{
							Name:  "force",
							Usage: "Force remove allocations off the node immediately",
						},
						cli.BoolFlag{
							Name:  "detach",
							Usage: "Return immediately instead of entering monitor mode",
						},
						cli.BoolFlag{
							Name:  "ignore-system",
							Usage: "Ignore system allows the drain to complete without stopping system job allocations. By default system jobs are stopped last.",
						},
						cli.BoolFlag{
							Name:  "keep-ineligible",
							Usage: "Keep ineligible will maintain the node's scheduling ineligibility even if the drain is being disabled. This is useful when an existing drain is being cancelled but additional scheduling on the node is not desired.",
						},
					},
					Action: func(c *cli.Context) error {
						if err := node.Drain(c); err != nil {
							log.Fatal(err)
							return err
						}

						return nil
					},
				},
				{
					Name:    "eligibility",
					Aliases: []string{"eligible"},
					Usage:   "The eligibility command is used to toggle scheduling eligibility for a given node. By default node's are eligible for scheduling meaning they can receive placements and run new allocations. Node's that have their scheduling elegibility disabled are ineligibile for new placements.",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "enable",
							Usage: "Enable scheduling eligbility",
						},
						cli.BoolFlag{
							Name:  "disable",
							Usage: "Disable scheduling eligibility",
						},
					},
					Action: func(c *cli.Context) error {
						if err := node.Eligibility(c); err != nil {
							log.Fatal(err)
							return err
						}

						return nil
					},
				},
			},
		},
		{
			Name:  "scale-export",
			Usage: "Export nomad job scale config to a local file",
			Action: func(c *cli.Context) error {
				configFile := c.Args().Get(0)
				if configFile == "" {
					return fmt.Errorf("Missing file name")
				}

				return scale.ExportCommand(configFile)
			},
		},
		{
			Name:  "scale-import",
			Usage: "Import nomad job scale config from a local file to Nomad cluster",
			Action: func(c *cli.Context) error {
				configFile := c.Args().Get(0)
				if configFile == "" {
					return fmt.Errorf("Missing file name")
				}

				return scale.ImportCommand(configFile)
			},
		},
		{
			Name:  "drain",
			Usage: "Drain node and wait for all allocations to stop",
			Action: func(c *cli.Context) error {
				return drain.App()
			},
		},
		{
			Name:  "reevaluate-all",
			Usage: "Force re-evaluate all jobs",
			Action: func(c *cli.Context) error {
				return reevaluate.App()
			},
		},
		{
			Name:  "gc",
			Usage: "Force a cluster GC",
			Action: func(c *cli.Context) error {
				return gc.App()
			},
		},
		{
			Name:  "firehose",
			Usage: "Firehose emit cluster changes",
			Action: func(c *cli.Context) error {
				return firehose.App()
			},
		},
	}
	app.Before = func(c *cli.Context) error {
		// convert the human passed log level into logrus levels
		level, err := log.ParseLevel(c.String("log-level"))
		if err != nil {
			log.Fatal(err)
		}
		log.SetLevel(level)

		return nil
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	app.Run(os.Args)
}
