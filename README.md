# cronus - Codefresh DockerHub Event Provider
[![Codefresh build status]( https://g.codefresh.io/api/badges/build?repoOwner=codefresh-io&repoName=cronus&branch=master&pipelineName=cronus&accountName=codefresh-inc&type=cf-1)]( https://g.codefresh.io/repositories/codefresh-io/cronus/builds?filter=trigger:build;branch:master;service:5a8999cf6b985f0001c6142b~cronus) [![Go Report Card](https://goreportcard.com/badge/github.com/codefresh-io/cronus)](https://goreportcard.com/report/github.com/codefresh-io/cronus) [![codecov](https://codecov.io/gh/codefresh-io/cronus/branch/master/graph/badge.svg)](https://codecov.io/gh/codefresh-io/cronus)

[![](https://images.microbadger.com/badges/image/codefresh/cronus.svg)](http://microbadger.com/images/codefresh/cronus) [![](https://images.microbadger.com/badges/commit/codefresh/cronus.svg)](https://microbadger.com/images/codefresh/cronus) [![Docker badge](https://img.shields.io/docker/pulls/codefresh/cronus.svg)](https://hub.docker.com/r/codefresh/cronus/)

Codefresh **Cron Event Provider**, code named *cronus*, sends (periodically, defined by `cron` expression) a short text message to the [Hermes](https://github.com/codefresh-io/hermes) trigger manager service.

## Cronus Normalized event

POST ${HERMES_SERVICE}/trigger/${event}

```json
{
    "secret": "<config secret>",
    "variables": {
        "message": "<config text message>",
        "timestamp": "<RFC3339 formated timestamp>"
    }
}
```

### Fields

- URL `event` - cronus event URI
- PAYLOAD `secret` - event secret
- PAYLOAD `variables` - set of variables
- PAYLOAD `variables:message` - event short text message (as specified when created)
- PAYLOAD `variables:timestamp` - event timestamp `{time RFC 3339}`

### Cronus Event URI

`cron:codefresh:{{cron-expression}}:{{message}}[:{{account}}]`

- `cron-expression '+'` - cron expression format (see below)
- `message` - message to be send with each cron trigger event; should be short and alpha-numeric only (no space characters); `[a-z0-9]+` regex
- `account` - optional Codefresh account short hash

#### URL Encoding

When using cron event URI with `cronus` REST API, make sure to apply URL encoding to it.

## CRON Expression Format

[CRON Expression Format](./docs/expression.md)

## Running cronus service

Run the `cronus server` command to start *cronus* CRON Event Provider.

```sh
NAME:
   cronus server - start cronus server

USAGE:
   cronus server [command options] [arguments...]

DESCRIPTION:
   Run Cronus CRON Event Provider server. Cronus generates time-based events and sends normalized event payload to the Codefresh Hermes trigger manager service to invoke associated Codefresh pipelines.

    Event URI Pattern: cron:codefresh:{{cron-expression}}:{{message}}

OPTIONS:
   --hermes value           Codefresh Hermes service (default: "http://hermes/") [$HERMES_SERVICE]
   --token value, -t value  Codefresh Hermes API token (default: "TOKEN") [$HERMES_TOKEN]
   --port value             TCP port for the dockerhub provider server (default: 8080)
   --dry-run                do not execute commands, just log
```

## Building cronus

`cronus` requires Go SDK to build.

1. Clone this repository into `$GOPATH/src/github.com/codefresh-io/cronus`
1. Run `hack/build.sh` helper script or `go build cmd/main.go`%
1. Run `hack/test.sh` to run all tests