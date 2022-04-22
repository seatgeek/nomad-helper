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
    - [namespace](#namespace)
        - [gc](#gc)
    - [node](#node)
        - [Filter examples](#filter-examples)
        - [drain](#drain)
            - [Examples](#examples)
        - [eligibility](#eligibility)
            - [Examples](#examples-1)
        - [breakdown](#breakdown)
        - [list](#list)
        - [discover](#discover)
    - [job](#job)
        - [stop](#stop)
        - [move](#move)
        - [hunt](#hunt)
    - [scale](#scale)
        - [export](#export)
        - [import](#import)
        - [Example Scale config](#example-scale-config)
    - [server](#server)
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

Automatically handle discovery of allocation and tail both `stdout` and `stderr` at the same time

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

## namespace

namespace specific commands

### gc

```
NAME:
   nomad-helper namespace gc - Cleans up empty namespaces

USAGE:
   nomad-helper namespace gc [command options] [arguments...]

OPTIONS:
   --dry  Dry run, just print actions
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
     eligibility, eligible  The eligibility command is used to toggle scheduling eligibility for a given node. By default node's are eligible for scheduling meaning they can receive placements and run new allocations. Node's that have their scheduling eligibility disabled are ineligible for new placements.

OPTIONS:
   --filter-prefix ef30d57c                                   Filter nodes by their ID with prefix matching ef30d57c
   --filter-class batch-jobs                                  Filter nodes by their node class batch-jobs
   --filter-version 0.8.4                                     Filter nodes by their Nomad version 0.8.4
   --filter-eligibility                                       Filter nodes by their scheduling eligibility
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
   --with-benefits    Instead of flipping the regular drain flag it will make instance ineligible and will add a desired constraint to the task groups found on the node
        --constraint
        --operand
        --value
        --wait-for-pending  will wait for all the moved jobs to reach running state
```

#### Examples

- `nomad-helper node drain --enable`
- `nomad-helper node --filter-class wrecker --filter-meta 'aws.ami-version=2.0.0-alpha14' --filter-meta 'aws.instance.availability-zone=us-east-1e' drain --noop --enable`
- `node --filter-meta "aws.ami-version=1.9.6" drain --enable --with-benefits --constraint meta.aws.ami-version --operand '=' --value 1.9.8 --wait-for-pending`

### eligibility

Filtering options can be found in the main `node` command help above

```
NAME:
   nomad-helper node eligibility - The eligibility command is used to toggle scheduling eligibility for a given node. By default node's are eligible for scheduling meaning they can receive placements and run new allocations. Node's that have their scheduling eligibility disabled are ineligible for new placements.

USAGE:
   nomad-helper node [filter options] eligibility [command options] [arguments...]

OPTIONS:
   --enable   Enable scheduling eligibility
   --disable  Disable scheduling eligibility
```

#### Examples

- `nomad-helper node eligibility --enable`
- `nomad-helper node --filter-class wrecker --filter-meta 'aws.ami-version=2.0.0-alpha14' --filter-meta 'aws.instance.availability-zone=us-east-1e' eligibility --enable`

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

### Discover

```
NAME:
   nomad-helper node discover - Output the Nomad client Meta and Attribute fields present in your cluster

USAGE:
   nomad-helper node [filters...] discover [command options]

OPTIONS:
   --output-format value  Either "table", "json" or "json-pretty" (default: "table")
```


## job

job specific commands

```
NAME:
   nomad-helper job - job specific commands with a twist

USAGE:
   nomad-helper job stop [command options] [arguments...]

COMMANDS:
     stop   Stop will stop the job with the ability to purge the job from the nomad cluster
     move   Move will add/append a constraint to the job that will resolve to moving the job
     hunt   Hunt will look for the jobs with discrepancy in job version between allocations

OPTIONS:
   --help, -h                                                 show help
```

### stop

```
USAGE:
   nomad-helper job stop [job_name] [command options]

OPTIONS:
   --as-prefix  Filter jobs by their name with prefix matching api-
   --dry        Only output jobs that would be stopped, don't do any modifications
```

#### Examples

- `nomad-helper job stop api`
- `nomad-helper job stop api --as-prefix`
- `nomad-helper job stop api --as-prefix --dry`


### move

```
USAGE:
   nomad-helper job stop [job_name] [command options]

OPTIONS:
   --as-prefix  Filter jobs by their name with prefix matching api-
   --constraint Constraint attribute
   --operand    Constraint operator
   --value      Constraint value
   --dry        Only output jobs that would be stopped, don't do any modifications
```

#### Examples

- `job move api --constraint meta.aws.ami-version --operand = --value 1.9.1 --exclude core`
- `job move api --as-prefix --constraint meta.aws.ami-version --operand = --value 1.9.1 --exclude core`

### hunt

```
USAGE:
   nomad-helper job hunt
```

#### Examples

- `job hunt`

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

## Server

```
NAME:
   nomad-helper server - Run a webserver exposing various endpoints

USAGE:
   nomad-helper server [command options] [arguments...]

DESCRIPTION:

  ** Arguments **

    * attribute.key will look up key in the "Attributes" Nomad client property
    * class / nodeclass for the Nomad client "NodeClass" property
    * datacenter / dc for the Nomad client "Datacenter" property
    * drain for the Nomad client "Drain" property
    * eligibility / schedulingeligibility for the Nomad client "SchedulingEligibility" property
    * hostname is an alias for attribute.unique.hostname
    * id for the Nomad client "ID" property
    * ip / address / ip-address is alias for attribute.unique.network.ip-address
    * meta.key will look up key in the "Meta" Nomad client configuration
    * name for the Nomad client "Name" property
    * status for the Nomad client "Status" property

  ** Filters **

  Filters are always passed as HTTP query arguments, order doesn't matter

    /?filter-attribute=driver.docker.version=17.09.0-ce        Filter nodes by their attribute key/value like 'driver.docker.version=17.09.0-ce'.
    /?filter-class=batch-jobs                                  Filter nodes by their node class batch-jobs
    /?filter-eligibility=eligible/ineligible                   Filter nodes by their eligibility status eligible/ineligible
    /?filter-meta=aws.instance.availability-zone=us-east-1e    Filter nodes by their meta key/value like 'aws.instance.availability-zone=us-east-1e'.
    /?filter-prefix=ef30d57c                                   Filter nodes by their ID with prefix matching ef30d57c
    /?filter-version=0.8.4                                     Filter nodes by their Nomad version 0.8.4

  ** Examples **

  Fields are always passed as HTTP path, and processed in order

    * /help
    * /help/node/breakdown
    * /help/node/list
    * /help/node/discover
    * /help/[command]/[subcommand]
    * /node/[breakdown|list]/class/status
    * /node/[breakdown|list]/meta.aws.instance.region/attribute.nomad.version
    * /node/[breakdown|list]/attribute.nomad.version/attribute.driver.docker


OPTIONS:
   --listen value  (default: "0.0.0.0:8000") [$LISTEN]
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
