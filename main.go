package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/seatgeek/nomad-helper/command/attach"
	"github.com/seatgeek/nomad-helper/command/gc"
	"github.com/seatgeek/nomad-helper/command/job"
	"github.com/seatgeek/nomad-helper/command/node"
	"github.com/seatgeek/nomad-helper/command/reevaluate"
	"github.com/seatgeek/nomad-helper/command/scale"
	"github.com/seatgeek/nomad-helper/command/server"
	"github.com/seatgeek/nomad-helper/command/tail"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
	"gopkg.in/workanator/go-ataman.v1"
)

var rndr = ataman.NewRenderer(ataman.BasicStyle())

var fieldHelpText = `
	<bold,underline>** Arguments **<reset>

		* <bold>attribute.<reset,underline>key<reset> will look up <underline>key<reset> in the "Attributes" Nomad client property
		* <bold>class<reset> / <bold>nodeclass<reset> for the Nomad client "NodeClass" property
		* <bold>datacenter<reset> / <bold>dc<reset> for the Nomad client "Datacenter" property
		* <bold>drain<reset> for the Nomad client "Drain" property
		* <bold>eligibility<reset> / <bold>schedulingeligibility<reset> for the Nomad client "SchedulingEligibility" property
		* <bold>hostname<reset> is an alias for <bold>attribute.<reset,underline>unique.hostname<reset>
		* <bold>id<reset> for the Nomad client "ID" property
		* <bold>ip<reset> / <bold>address<reset> / <bold>ip-address<reset> is alias for <bold>attribute.<reset,underline>unique.network.ip-address<reset>
		* <bold>meta.<reset,underline>key<reset> will look up <underline>key<reset> in the "Meta" Nomad client configuration
		* <bold>name<reset> for the Nomad client "Name" property
		* <bold>status<reset> for the Nomad client "Status" property
`

var filterHelpText = `
	<bold,underline>** Filters **<reset>

		--filter-attribute 'driver.docker.version=17.09.0-ce'      Filter nodes by their attribute key/value like 'driver.docker.version=17.09.0-ce'. Flag can be repeated.
		--filter-class batch-jobs                                  Filter nodes by their node class batch-jobs
		--filter-eligibility eligible/ineligible                   Filter nodes by their eligibility status eligible/ineligible
		--filter-meta 'aws.instance.availability-zone=us-east-1e'  Filter nodes by their meta key/value like 'aws.instance.availability-zone=us-east-1e'. Flag can be repeated.
		--filter-prefix ef30d57c                                   Filter nodes by their ID with prefix matching ef30d57c
		--filter-version 0.8.4                                     Filter nodes by their Nomad version 0.8.4
`

var filterWebHelpText = `
	<bold,underline>** Filters **<reset>

	Filters are always passed as HTTP query arguments, order doesn't matter

		/?filter-attribute=driver.docker.version=17.09.0-ce        Filter nodes by their attribute key/value like 'driver.docker.version=17.09.0-ce'.
		/?filter-class=batch-jobs                                  Filter nodes by their node class batch-jobs
		/?filter-eligibility=eligible/ineligible                   Filter nodes by their eligibility status eligible/ineligible
		/?filter-meta=aws.instance.availability-zone=us-east-1e    Filter nodes by their meta key/value like 'aws.instance.availability-zone=us-east-1e'.
		/?filter-prefix=ef30d57c                                   Filter nodes by their ID with prefix matching ef30d57c
		/?filter-version=0.8.4                                     Filter nodes by their Nomad version 0.8.4
`

var helpExamples = `
	<bold,underline>** Examples **<reset>

		* nomad-helper node __COMMAND__ <bold>class status<reset>
		* nomad-helper node __COMMAND__ <bold>attribute<reset,underline>.nomad.version<reset,bold> attribute.<reset,underline>driver.docker<reset>
		* nomad-helper node __COMMAND__ <bold>meta.<reset,underline>aws.instance.region<reset,bold> attribute.<reset,underline>nomad.version<reset>
`

var helpWebExamples = `
	<bold,underline>** Examples **<reset>

	Fields are always passed as HTTP path, and processed in order

		* /help
		* /help/node/breakdown
		* /help/node/list
		* /help/node/discover
		* /help/[command]/[subcommand]
		* /node/[breakdown|list]/<bold>class<reset>/<bold>status<reset>
		* /node/[breakdown|list]/<bold>meta.<reset,underline>aws.instance.region<reset>/<bold>attribute.<reset,underline>nomad.version<reset>
		* /node/[breakdown|list]/<bold>attribute<reset,underline>.nomad.version<reset>/<bold>attribute.<reset,underline>driver.docker<reset>
`

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

// Version is filled in by the compiler (git tag + changes)
var Version = "local-dev"

func main() {
	app := cli.NewApp()
	app.Name = "nomad-helper"
	app.Usage = "Useful utilties for working with Nomad at scale"
	app.Version = Version

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
				err := attach.Run(c)
				if err != nil {
					log.Fatal(err)
				}

				return err
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
				err := tail.Run(c)
				if err != nil {
					log.Fatal(err)
				}

				return err
			},
		},
		{
			Name:  "job",
			Usage: "job specific commands with a twist (see help)",
			Flags: filterFlags,
			Subcommands: []cli.Command{
				{
					Name:  "stop",
					Usage: "Stop jobs in the cluster",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "purge",
							Usage: "Purge job",
						},
						cli.BoolFlag{
							Name:  "dry",
							Usage: "Dry run, just print actions",
						},
						cli.BoolFlag{
							Name:  "as-prefix",
							Usage: "Treat the job name as a job prefix (job name 'api-' would mean all jobs 'api-*' would be stopped)",
						},
					},
					Action: func(c *cli.Context) error {
						err := job.Stop(c, log.StandardLogger())
						if err != nil {
							log.Fatal(err)
						}

						return err
					},
				},
				{
					Name:  "move",
					Usage: "Move jobs in the cluster",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "dry",
							Usage: "Dry run, just print actions",
						},
						cli.BoolFlag{
							Name:  "as-prefix",
							Usage: "Treat the job name as a job prefix (job name 'api-' would mean all jobs 'api-*' would be stopped)",
						},
						cli.StringFlag{
							Name:  "exclude",
							Usage: "Filter out jobs with substring in name",
						},
						cli.StringFlag{
							Name:  "constraint",
							Usage: "Constraint attribute",
						},
						cli.StringFlag{
							Name:  "operand",
							Usage: "operator",
						},
						cli.StringFlag{
							Name:  "value",
							Usage: "value of constraint to check",
						},
					},
					Action: func(c *cli.Context) error {
						err := job.Move(c, log.StandardLogger())
						if err != nil {
							log.Fatal(err)
						}

						return err
					},
				},
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
						cli.BoolFlag{
							Name:  "with-benefits",
							Usage: "Instead of draining the node in a regular way move the jobs to specific constraints",
						},
						cli.StringFlag{
							Name:  "constraint",
							Usage: "Constraint attribute",
						},
						cli.StringFlag{
							Name:  "operand",
							Usage: "operator",
						},
						cli.StringFlag{
							Name:  "value",
							Usage: "value of constraint to check",
						},
						cli.BoolFlag{
							Name:  "wait-for-pending",
							Usage: "Will wait for pending allocation and blocked evaluations per job",
						},
					},
					Action: func(c *cli.Context) error {
						err := node.Drain(c, log.StandardLogger())
						if err != nil {
							log.Fatal(err)
						}

						return err
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
						err := node.Eligibility(c, log.StandardLogger())
						if err != nil {
							log.Fatal(err)
						}

						return err
					},
				},
				{
					Name:        "list",
					Usage:       `Output list of key properties for a Nomad client`,
					UsageText:   "nomad-helper node [filters...] list [command options] [keys...]",
					Description: rndr.MustRender(fieldHelpText) + rndr.MustRender(filterHelpText) + rndr.MustRender(strings.ReplaceAll(helpExamples, "__COMMAND__", "list")),
					ArgsUsage:   "[keys...]",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "output-format",
							Value: "table",
							Usage: "Either `table, json or json-pretty`",
						},
					},
					Action: func(c *cli.Context) error {
						err := node.ListCLI(c, log.StandardLogger())
						if err != nil {
							log.Fatal(err)
						}

						return err
					},
				},
				{
					Name:        "breakdown",
					Usage:       `Break down (count) how many Nomad clients that match a list of key properties`,
					UsageText:   "nomad-helper node [filters...] breakdown [command options] [keys...]",
					Description: rndr.MustRender(fieldHelpText) + rndr.MustRender(filterHelpText) + rndr.MustRender(strings.ReplaceAll(helpExamples, "__COMMAND__", "breakdown")),
					ArgsUsage:   "[keys...]",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "output-format",
							Value: "table",
							Usage: `Either "table", "json" or "json-pretty"`,
						},
					},
					Action: func(c *cli.Context) error {
						err := node.BreakdownCLI(c, log.StandardLogger())
						if err != nil {
							log.Fatal(err)
						}

						return err
					},
				},
				{
					Name:      "discover",
					Usage:     `Output the Nomad client Meta and Attribute fields present in your cluster`,
					UsageText: "nomad-helper node [filters...] discover [command options]",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "output-format",
							Value: "table",
							Usage: `Either "table", "json" or "json-pretty"`,
						},
					},
					Action: func(c *cli.Context) error {
						err := node.DiscoverCLI(c, log.StandardLogger())
						if err != nil {
							log.Fatal(err)
						}

						return err
					},
				},
				{
					Name:      "empty",
					Usage:     `List nodes that only have system jobs running`,
					UsageText: "nomad-helper node [filters...] empty [command options]",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "output-format",
							Value: "table",
							Usage: `Either "table", "json" or "json-pretty"`,
						},
					},
					Action: func(c *cli.Context) error {
						err := node.Empty(c, log.StandardLogger())
						if err != nil {
							log.Fatal(err)
						}

						return err
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
							return fmt.Errorf("Missing file name")
						}

						err := scale.ExportCommand(configFile)
						if err != nil {
							log.Fatal(err)
						}

						return err
					},
				},
				{
					Name:  "import",
					Usage: "Import nomad job scale config from a local file to Nomad cluster",
					Action: func(c *cli.Context) error {
						configFile := c.Args().Get(0)
						if configFile == "" {
							return fmt.Errorf("Missing file name")
						}

						err := scale.ImportCommand(configFile)
						if err != nil {
							log.Fatal(err)
						}

						return err
					},
				},
			},
		},
		{
			Name:   "stats",
			Hidden: true,
			Usage:  "Deprecated!",
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
				err := node.Stats(c)
				log.Error("")
				log.Error("'nomad-helper stats' is deprecated, please use 'nomad-helper node breakdown' instead")
				log.Error("")
				log.Error("Below is a best-effort compatible command for the migration")
				log.Error("")
				log.Error(err.Error())
				log.Fatal("")
				return err
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
			Name:        "server",
			Usage:       "Run a webserver exposing various endpoints",
			Description: rndr.MustRender(fieldHelpText) + rndr.MustRender(filterWebHelpText) + rndr.MustRender(strings.ReplaceAll(helpWebExamples, "__COMMAND__", "breakdown")),
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "listen",
					Value:  "0.0.0.0:8000",
					EnvVar: "LISTEN",
				},
			},
			Action: func(c *cli.Context) error {
				return server.Run(app, c, log.StandardLogger())
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
