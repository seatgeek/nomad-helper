# nomad-helper

- [nomad-helper](#nomad-helper)
- [Running](#running)
- [Requirements](#requirements)
- [Building](#building)
- [Configuration](#configuration)
- [Installation](#installation)
    - [Binary](#binary)
    - [Source](#source)
- [Usage](#usage)
    - [attach](#attach)
    - [tail](#tail)
    - [node](#node)
        - [Filter examples](#filter-examples)
        - [drain](#drain)
            - [Examples](#examples)
        - [eligibility](#eligibility)
            - [Examples](#examples-1)
        - [breakdown](#breakdown)
        - [list](#list)
    - [scale](#scale)
        - [export](#export)
        - [import](#import)
        - [Example Scale config](#example-scale-config)
    - [reevaluate-all](#reevaluate-all)
    - [gc](#gc)

`nomad-helper` is a tool meant to enable teams to quickly onboard themselves with nomad, by exposing scaling functionality in a simple to use and share yaml format.

# Running

The project has build artifacts for linux, darwin and windows in the [GitHub releases tab](https://github.com/seatgeek/nomad-helper/releases).

A docker container is also provided at [seatgeek/nomad-helper](https://hub.docker.com/r/seatgeek/nomad-helper/tags/)

# Requirements

- Go 1.13.1

# Building

Recommend environment variables

```sh
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org
```

To build a binary, run the following

```shell
# get this repo
go get github.com/seatgeek/nomad-helper

# go to the repo directory
cd $GOPATH/src/github.com/seatgeek/nomad-helper

# install the `nomad-helper` binary into $GOPATH/bin
make install

# install the `nomad-helper` binary ./build/nomad-helper-${GOOS}-${GOARCH}
make install
```

This will create a `nomad-helper` binary in your `$GOPATH/bin` directory.

# Configuration

Any `NOMAD_*` env that the native `nomad` CLI tool supports are supported by this tool.

The most basic requirement is `export NOMAD_ADDR=http://<ip>:4646`.

# Installation

## Binary

- Download the binary from [the release page](https://github.com/seatgeek/nomad-helper/releases)
- Use docker (`seatgeek/nomad-helper`)

## Source

- make sure Go 1.10+ is installed (`brew install go`)
- clone the repo into `$GOPATH/src/github.com/seatgeek/nomad-helper`
- make install
- `go install` / `go build` / `make build`

# Usage

The `nomad-helper` binary has several helper subcommands.

## attach

Automatically handle discovery of allocation host IP by CLI filters or interactive shell and ssh + attach to the running container.

The tool assume you can SSH to the instance and your `~/.ssh/config` is configured with the right configuration for doing so.

```
NAME:
   nomad-helper attach - attach to a specific allocation

USAGE:
   nomad-helper attach [command options] [arguments...]

OPTIONS:
   --job value      List allocations for the job and attach to the selected allocation
   --alloc value    Partial UUID or the full 36 char UUID to attach to
   --task value     Task name to auto-select if the allocation has multiple tasks in the allocation group
   --host           Connect to the host directly instead of attaching to a container
   --command value  Command to run when attaching to the container (default: "bash")
```

## tail

Automatically handle discovery of allocation and tail both `stdout` and `stdout` at the same time

```
NAME:
   nomad-helper tail - tail stdout/stderr from nomad alloc

USAGE:
   nomad-helper tail [command options] [arguments...]

OPTIONS:
   --job value    (optional) list allocations for the job and attach to the selected allocation
   --alloc value  (optional) partial UUID or the full 36 char UUID to attach to
   --task value   (optional) the task name to auto-select if the allocation has multiple tasks in the allocation group
   --stderr       (optional, default: true) tail stderr from nomad
   --stdout       (optional, default: true) tail stdout from nomad
```

## node

node specific commands that act on all Nomad clients that match the filters provided, rather than a single node

```
NAME:
   nomad-helper node - node specific commands that act on all Nomad clients that match the filters provided, rather than a single node

USAGE:
   nomad-helper node command [command options] [arguments...]

COMMANDS:
     drain                  The node drain command is used to toggle drain mode on a given node. Drain mode prevents any new tasks from being allocated to the node, and begins migrating all existing allocations away
     eligibility, eligible  The eligibility command is used to toggle scheduling eligibility for a given node. By default node's are eligible for scheduling meaning they can receive placements and run new allocations. Node's that have their scheduling elegibility disabled are ineligibile for new placements.

OPTIONS:
   --filter-prefix ef30d57c                                   Filter nodes by their ID with prefix matching ef30d57c
   --filter-class batch-jobs                                  Filter nodes by their node class batch-jobs
   --filter-version 0.8.4                                     Filter nodes by their Nomad version 0.8.4
   --filter-eligibility                                       Filter nodes by theit sheduling eligibility
   --percent                                                  Filter only specific percent of nodes
   --filter-meta 'aws.instance.availability-zone=us-east-1e'  Filter nodes by their meta key/value like 'aws.instance.availability-zone=us-east-1e'. Can be provided multiple times.
   --filter-attribute 'driver.docker.version=17.09.0-ce'      Filter nodes by their attribute key/value like 'driver.docker.version=17.09.0-ce'. Can be provided multiple times.
   --noop                                                     Only output nodes that would be drained, don't do any modifications
   --help, -h                                                 show help
```

### Filter examples

- `nomad-helper node <command> <args>`
- `nomad-helper node --noop --filter-meta 'aws.instance.availability-zone=us-east-1e'  --filter-attribute 'driver.docker.version=17.09.0-ce' <command> <args>`

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

#### Examples

- `nomad-helper node drain --enable`
- `nomad-helper node --filter-class wrecker --filter-meta 'aws.ami-version=2.0.0-alpha14' --filter-meta 'aws.instance.availability-zone=us-east-1e' drain --noop --enable`

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

#### Examples

- `nomad-helper node eligibility --enable`
- `nomad-helper node --filter-class wrecker --filter-meta 'aws.ami-version=2.0.0-alpha14' --filter-meta 'aws.instance.availability-zone=us-east-1e' eligibility --noop --enable`

### Breakdown

```
NAME:
   nomad-helper node breakdown - Break down (count) how many Nomad clients that match a list of key properties

USAGE:
   nomad-helper node [filters...] breakdown [command options] [keys...]

DESCRIPTION:

  ** Arguments **

    * attribute.key will look up key in the "Attributes" Nomad client property
    * hostname is an alias for attribute.unique.hostname
    * ip / address / ip-address is alias for attribute.unique.network.ip-address
    * meta.key will look up key in the "Meta" Nomad client configuration
    * class / nodeclass for the Nomad client "NodeClass" property
    * id for the Nomad client "ID" property
    * datacenter / dc for the Nomad client "Datacenter" property
    * drain for the Nomad client "Drain" property
    * status for the Nomad client "Status" property
    * eligibility / schedulingeligibility for the Nomad client "SchedulingEligibility" property

  ** Filters **

    --filter-prefix ef30d57c                                   Filter nodes by their ID with prefix matching ef30d57c
    --filter-class batch-jobs                                  Filter nodes by their node class batch-jobs
    --filter-version 0.8.4                                     Filter nodes by their Nomad version 0.8.4
    --filter-eligibility eligible/ineligible                   Filter nodes by their eligibility status eligible/ineligible
    --filter-meta 'aws.instance.availability-zone=us-east-1e'  Filter nodes by their meta key/value like 'aws.instance.availability-zone=us-east-1e'. Flag can be repeated.
    --filter-attribute 'driver.docker.version=17.09.0-ce'      Filter nodes by their attribute key/value like 'driver.docker.version=17.09.0-ce'. Flag can be repeated.

  ** Examples **

    * nomad-helper node breakdown class status
    * nomad-helper node breakdown attribute.nomad.version attribute.driver.docker
    * nomad-helper node breakdown meta.aws.instance.region attribute.nomad.version


OPTIONS:
   --output-format value  Either "table", "json" or "json-pretty" (default: "table")
```

### List

```
NAME:
   nomad-helper node list - Output list of key properties for a Nomad client

USAGE:
   nomad-helper node [filters...] list [command options] [keys...]

DESCRIPTION:

  ** Arguments **

    * attribute.key will look up key in the "Attributes" Nomad client property
    * hostname is an alias for attribute.unique.hostname
    * ip / address / ip-address is alias for attribute.unique.network.ip-address
    * meta.key will look up key in the "Meta" Nomad client configuration
    * class / nodeclass for the Nomad client "NodeClass" property
    * id for the Nomad client "ID" property
    * datacenter / dc for the Nomad client "Datacenter" property
    * drain for the Nomad client "Drain" property
    * status for the Nomad client "Status" property
    * eligibility / schedulingeligibility for the Nomad client "SchedulingEligibility" property

  ** Filters **

    --filter-prefix ef30d57c                                   Filter nodes by their ID with prefix matching ef30d57c
    --filter-class batch-jobs                                  Filter nodes by their node class batch-jobs
    --filter-version 0.8.4                                     Filter nodes by their Nomad version 0.8.4
    --filter-eligibility eligible/ineligible                   Filter nodes by their eligibility status eligible/ineligible
    --filter-meta 'aws.instance.availability-zone=us-east-1e'  Filter nodes by their meta key/value like 'aws.instance.availability-zone=us-east-1e'. Flag can be repeated.
    --filter-attribute 'driver.docker.version=17.09.0-ce'      Filter nodes by their attribute key/value like 'driver.docker.version=17.09.0-ce'. Flag can be repeated.

  ** Examples **

    * nomad-helper node list class status
    * nomad-helper node list attribute.nomad.version attribute.driver.docker
    * nomad-helper node list meta.aws.instance.region attribute.nomad.version


OPTIONS:
   --output-format table, json or json-pretty  Either table, json or json-pretty (default: "table")
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

`nomad-helper scale export production.yml` will read the Nomad cluster `job  + group + count` values and write them to a local `production.yml` file.

```
NAME:
   nomad-helper scale export - Export nomad job scale config to a local file from Nomad cluster

USAGE:
   nomad-helper scale export [arguments...]
```

### import

`nomad-helper scale import production.yml` will update the Nomad cluster `job + group + count` values according to the values in a local `production.yaml` file.

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
