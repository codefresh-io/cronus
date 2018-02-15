# cronus - Codefresh DockerHub Event Provider

[![Go Report Card](https://goreportcard.com/badge/github.com/codefresh-io/cronus)](https://goreportcard.com/report/github.com/codefresh-io/cronus) [![codecov](https://codecov.io/gh/codefresh-io/cronus/branch/master/graph/badge.svg)](https://codecov.io/gh/codefresh-io/cronus)

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

- URL `event` - event URI in form `cron:codefresh:{min.exp}-{hour.exp}-{day.exp}-{month.exp}-{day.week.exp}`; `XXX.exp` is a URL friendly `cron` expression, see replacement table below
- PAYLOAD `secret` - event secret
- PAYLOAD `variables` - set of variables
- PAYLOAD `variables:message` - event short text message (as specified when created)
- PAYLOAD `variables:timestamp` - event timestamp `{time RFC 3339}`

## CRON Expression Format

[CRON Expression Format](./docs/expression.md)

## Running cronus service

Run the `cronus server` command to start *cronus* CRON Event Provider.

```sh
NAME:
   cronus server - start cronus CRON event provider server

USAGE:
   cronus server [command options] [arguments...]

DESCRIPTION:
   Run Cronus CRON Event Provider server. Cronus generates time-based events and sends normalized event payload to the Codefresh Hermes trigger manager service to invoke associated Codefresh pipelines.

OPTIONS:
   --hermes value, --hm value  Codefresh Hermes service (default: "http://hermes/") [$HERMES_SERVICE]
   --token value, -t value     Codefresh Hermes API token (default: "TOKEN") [$HERMES_TOKEN]
```

## Building cronus

`cronus` requires Go SDK to build.

1. Clone this repository into `$GOPATH/src/github.com/codefresh-io/cronus`
1. Run `hack/build.sh` helper script or `go build cmd/main.go`%
1. Run `hack/test.sh` to run all tests