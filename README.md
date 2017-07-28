# nomad-helper

`nomad-helper` is a tool meant to enable teams to quickly onboard themselves with nomad, by exposing scaling functionality in a simple to use and share yaml format.

## Running

The project got build artifacts for linux, darwin and windows in the [GitHub releases tab](https://github.com/seatgeek/nomad-helper/releases).

A docker container is also provided at [seatgeek/nomad-helper](https://hub.docker.com/r/seatgeek/nomad-helper/tags/)

## Requirements

- Go 1.8
- govender https://github.com/kardianos/govendor/

## Building

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

## Configuration

Any `NOMAD_*` env that the native `nomad` CLI tool supports are supported by this tool.

The most basic requirement is `export NOMAD_ADDR=http://<ip>:4646`.

## Usage

The `nomad-helper` binary has several helper subcommands.

### `reevaluate-all`

`nomad-helper reevaluate-all` will force re-evaluate all jobs in the cluster. This will cause failed or lost allocations to be put back into the cluster.

### `drain`

`nomad-helper drain` will dr1ain the node and block until all allocations no longer have "running" or "pending" state.

The node to be drained is specified via the `$NOMAD_ADDR` environment variable.

### `gc`

`nomad-helper gc` will force a cluster garbage collection.

### `scale-export`

`nomad-helper scale-export production.yml` will read the Nomad cluster `job  + group + count` values and write them to a local `production.yml` file.

The Nomad cluster is specified via the `$NOMAD_ADDR` environment variable.

### `scale-import`

`nomad-helper scale-import production.yml` will update the Nomad cluster `job + group + count` values according to the values in a local `production.yaml` file.

The Nomad cluster is specified via the `$NOMAD_ADDR` environment variable.

## Example Scale config

```
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
