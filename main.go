package main

import (
	"os"
	"sort"

	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/seatgeek/nomad-helper/command/drain"
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
