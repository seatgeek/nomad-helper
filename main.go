package main

import (
	"os"
	"sort"
	"time"

	"github.com/seatgeek/nomad-helper/command/attach"
	"github.com/seatgeek/nomad-helper/command/gc"
	"github.com/seatgeek/nomad-helper/command/node"
	"github.com/seatgeek/nomad-helper/command/reevaluate"
	"github.com/seatgeek/nomad-helper/command/scale"
	"github.com/seatgeek/nomad-helper/command/stats"
	"github.com/seatgeek/nomad-helper/command/tail"
	log "github.com/sirupsen/logrus"
	cli "gopkg.in/urfave/cli.v1"
)

var filterFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "filter-prefix",
		Usage: "Filter nodes by their ID with prefix matching `ef30d57c`",
	},
	cli.StringFlag{
		Name:  "filter-class",
		Usage: "Filter nodes by their node class `batch-jobs`",
	},
	cli.StringFlag{
		Name:  "filter-version",
		Usage: "Filter nodes by their Nomad version `0.8.4`",
	},
	cli.StringFlag{
		Name:  "filter-eligibility",
		Usage: "Filter nodes by their eligibility status `eligible/ineligible`",
	},
	cli.IntFlag{
		Name:  "percent",
		Usage: "Filter only specific percent of nodes percent of nodes",
		Value: 100,
	},
	cli.StringSliceFlag{
		Name:  "filter-meta",
		Usage: "Filter nodes by their meta key/value like `'aws.instance.availability-zone=us-east-1e'`. Can be provided multiple times.",
	},
	cli.StringSliceFlag{
		Name:  "filter-attribute",
		Usage: "Filter nodes by their attribute key/value like `'driver.docker.version=17.09.0-ce'`. Can be provided multiple times.",
	},
	cli.BoolFlag{
		Name:  "noop",
		Usage: "Only output nodes that would be drained, don't do any modifications",
	},
	cli.BoolFlag{
		Name:  "no-progress",
		Usage: "Do not show progress bar",
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "nomad-helper"
	app.Usage = "Useful utilties for working with Nomad at scale"
	app.Version = "1.0"

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
			Name:  "attach",
			Usage: "attach to a specific allocation",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "job",
					Usage: "List allocations for the job and attach to the selected allocation",
				},
				cli.StringFlag{
					Name:  "alloc",
					Usage: "Partial UUID or the full 36 char UUID to attach to",
				},
				cli.StringFlag{
					Name:  "task",
					Usage: "Task name to auto-select if the allocation has multiple tasks in the allocation group",
				},
				cli.BoolFlag{
					Name:  "host",
					Usage: "Connect to the host directly instead of attaching to a container",
				},
				cli.StringFlag{
					Name:  "command",
					Value: "bash",
					Usage: "Command to run when attaching to the container",
				},
			},
			Action: func(c *cli.Context) error {
				if err := attach.Run(c); err != nil {
					log.Fatal(err)
					return err
				}

				return nil
			},
		},
		{
			Name:  "tail",
			Usage: "tail stdout/stderr from nomad alloc",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "job",
					Usage: "(optional) list allocations for the job and attach to the selected allocation",
				},
				cli.StringFlag{
					Name:  "alloc",
					Usage: "(optional) partial UUID or the full 36 char UUID to attach to",
				},
				cli.StringFlag{
					Name:  "task",
					Usage: "(optional) the task name to auto-select if the allocation has multiple tasks in the allocation group",
				},
				cli.BoolTFlag{
					Name:  "stderr",
					Usage: "(optional, default: true) tail stderr from nomad",
				},
				cli.BoolTFlag{
					Name:  "stdout",
					Usage: "(optional, default: true) tail stdout from nomad",
				},
				cli.StringFlag{
					Name:  "writer",
					Value: "color",
					Usage: "(optional, default: color) writer type (raw, color, simple)",
				},
				cli.StringFlag{
					Name:  "theme, ct",
					Value: "emacs",
					Usage: "(optional, default: emacs) Chroma color scheme to use - see https://xyproto.github.io/splash/docs/",
				},
			},
			Action: func(c *cli.Context) error {
				if err := tail.Run(c); err != nil {
					log.Fatal(err)
					return err
				}

				return nil
			},
		},
		{
			Name:  "node",
			Usage: "node specific commands that act on all Nomad clients that match the filters provided, rather than a single node",
			Flags: filterFlags,
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
						cli.BoolFlag{
							Name:  "no-progress",
							Usage: "Do not show progress bar",
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
			Name:  "scale",
			Usage: "Import / Export job -> group -> count values",
			Subcommands: []cli.Command{
				{
					Name:  "export",
					Usage: "Export nomad job scale config to a local file from Nomad cluster",
					Action: func(c *cli.Context) error {
						configFile := c.Args().Get(0)
						if configFile == "" {
							log.Fatal("Missing file name")
						}

						if err := scale.ExportCommand(configFile); err != nil {
							log.Fatal(err)
						}

						return nil
					},
				},
				{
					Name:  "import",
					Usage: "Import nomad job scale config from a local file to Nomad cluster",
					Action: func(c *cli.Context) error {
						configFile := c.Args().Get(0)
						if configFile == "" {
							log.Fatal("Missing file name")
						}

						if err := scale.ImportCommand(configFile); err != nil {
							log.Fatal(err)
						}

						return nil
					},
				},
			},
		},
		{
			Name:  "stats",
			Usage: "Get cluster stats",
			Flags: append(filterFlags,
				cli.StringSliceFlag{
					Name: "dimension",
				},
				cli.StringFlag{
					Name:  "output-format",
					Value: "table",
					Usage: "Either `table, json or json-pretty`",
				},
			),
			Action: func(c *cli.Context) error {
				if err := stats.Run(c); err != nil {
					log.Error(err)
					return err
				}
				return nil
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
	sort.Sort(cli.CommandsByName(app.Commands))
	app.Run(os.Args)
}
