# nomad-helper

## Running

The project got build artifacts for linux, darwin and windows in the [GitHub releases tab](https://github.com/seatgeek/nomad-helper/releases).

A docker container is also provided at [seatgeek/nomad-helper](https://hub.docker.com/r/seatgeek/nomad-helper/tags/)

## Configuration

Any `NOMAD_*` env that the native `nomad` CLI tool support, is supported by this tool.

The most basic requirement is `export NOMAD_ADDR=http://<ip>:4646`.

## Drain

`noamd-helper drain` will drain the node, and block until all allocations no longer have "running" or "pending" state.

## Scale Export

`nomad-helper scale-export perf.yml` will write the remote Nomad Cluster (from `$NOMAD_ADDR` env) `job  + group + count` values to to `perf.yml`

## Scale Import

`nomad-helper scale-import perf.yml` will change the Remote Nomad Cluster `job + group + count` values according to the values in `perf.yaml`

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
