# nomad-helper

<!-- TOC -->

- [nomad-helper](#nomad-helper)
- [Running](#running)
- [Requirements](#requirements)
- [Building](#building)
- [Configuration](#configuration)
- [Usage](#usage)
    - [node](#node)
        - [drain](#drain)
        - [eligibility](#eligibility)
    - [scale](#scale)
        - [export](#export)
        - [import](#import)
        - [Example Scale config](#example-scale-config)
    - [reevaluate-all](#reevaluate-all)
    - [gc](#gc)

<!-- /TOC -->

`nomad-helper` is a tool meant to enable teams to quickly onboard themselves with nomad, by exposing scaling functionality in a simple to use and share yaml format.

# Running

The project got build artifacts for linux, darwin and windows in the [GitHub releases tab](https://github.com/seatgeek/nomad-helper/releases).

A docker container is also provided at [seatgeek/nomad-helper](https://hub.docker.com/r/seatgeek/nomad-helper/tags/)

# Requirements

- Go 1.10
- govender https://github.com/kardianos/govendor/

# Building

To build a binary, run the following

```shell
# get this repo
go get github.com/seatgeek/nomad-helper

# go to the repo directory
cd $GOPATH/src/github.com/seatgeek/nomad-helper

# build the `nomad-helper` binary
make build
```

This will create a `nomad-helper` binary in your `$GOPATH/bin` directory.

# Configuration

Any `NOMAD_*` env that the native `nomad` CLI tool supports are supported by this tool.

The most basic requirement is `export NOMAD_ADDR=http://<ip>:4646`.

# Usage

The `nomad-helper` binary has several helper subcommands.

## node

node specific commands that act on all Nomad clients that match the filters provided, rather than a single node

```
NAME:
   nomad-helper node - node specific commands that act on all Nomad clients that match the filters provided, rather than a single node

USAGE:
   nomad-helper node [filter options] command [command options] [arguments...]

COMMANDS:
     drain                  The node drain command is used to toggle drain mode on a given node. Drain mode prevents any new tasks from being allocated to the node, and begins migrating all existing allocations away
     eligibility, eligible  The eligibility command is used to toggle scheduling eligibility for a given node. By default node's are eligible for scheduling meaning they can receive placements and run new allocations. Node's that have their scheduling elegibility disabled are ineligibile for new placements.

OPTIONS:
   --filter-prefix value         Filter nodes by their ID (prefix matching)
   --filter-class value          Filter nodes by their node class
   --filter-nomad-version value  Filter nodes by their Nomad version
   --filter-ami-version value    Filter nodes by their Instance AMI version (BaseAMI)
   --noop                        Only output nodes that would be drained, don't do any modifications
   --help, -h                    show help
```

### drain

Filtering options can be found in the main `node` command help above

```
USAGE:
   nomad-helper node [filter options] drain [command options] [arguments...]

OPTIONS:
   --enable           Enable node drain mode
   --disable          Disable node drain mode
   --deadline value   Set the deadline by which all allocations must be moved off the node. Remaining allocations after the deadline are force removed from the node. Defaults to 1 hour (default: 1h0m0s)
   --no-deadline      No deadline allows the allocations to drain off the node without being force stopped after a certain deadline
   --monitor          Enter monitor mode directly without modifying the drain status
   --force            Force remove allocations off the node immediately
   --detach           Return immediately instead of entering monitor mode
   --ignore-system    Ignore system allows the drain to complete without stopping system job allocations. By default system jobs are stopped last.
   --keep-ineligible  Keep ineligible will maintain the node's scheduling ineligibility even if the drain is being disabled. This is useful when an existing drain is being cancelled but additional scheduling on the node is not desired.
```

### eligibility

Filtering options can be found in the main `node` command help above

```
NAME:
   nomad-helper node eligibility - The eligibility command is used to toggle scheduling eligibility for a given node. By default node's are eligible for scheduling meaning they can receive placements and run new allocations. Node's that have their scheduling elegibility disabled are ineligibile for new placements.

USAGE:
   nomad-helper node [filter options] eligibility [command options] [arguments...]

OPTIONS:
   --enable   Enable scheduling eligbility
   --disable  Disable scheduling eligibility
```

## scale

```
NAME:
   nomad-helper scale - Import / Export job -> group -> count values

USAGE:
   nomad-helper scale command [command options] [arguments...]

COMMANDS:
     export  Export nomad job scale config to a local file from Nomad cluster
     import  Import nomad job scale config from a local file to Nomad cluster

OPTIONS:
   --help, -h  show help
```

### export

`nomad-helper scale-export production.yml` will read the Nomad cluster `job  + group + count` values and write them to a local `production.yml` file.

```
NAME:
   nomad-helper scale export - Export nomad job scale config to a local file from Nomad cluster

USAGE:
   nomad-helper scale export [arguments...]
```

### import

`nomad-helper scale-import production.yml` will update the Nomad cluster `job + group + count` values according to the values in a local `production.yaml` file.

```
NAME:
   nomad-helper scale import - Import nomad job scale config from a local file to Nomad cluster

USAGE:
   nomad-helper scale import [arguments...]
```

### Example Scale config

```yml
info:
  exported_at: Thu, 29 Jun 2017 13:11:19 +0000
  exported_by: jippi
  nomad_addr: http://nomad.service.consul:4646
jobs:
  nginx:
    server: 10
  api-es:
    api-es-1: 1
    api-es-2: 1
    api-es-3: 1
```


## reevaluate-all

```
NAME:
   nomad-helper reevaluate-all - Force re-evaluate all jobs

USAGE:
   nomad-helper reevaluate-all [arguments...]
```

## gc

```
NAME:
   nomad-helper gc - Force a cluster GC

USAGE:
   nomad-helper gc [arguments...]
```
