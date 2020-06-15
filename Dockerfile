# Copyright 2015 The Kubernetes Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# ----- Go Dev Image ------
#
FROM golang:1.11 AS godev

# set working directory
RUN mkdir -p /go/src/github.com/codefresh-io/cronus
WORKDIR /go/src/github.com/codefresh-io/cronus

# copy sources
COPY . .

ENV GO111MODULE=on
RUN go mod tidy

#
# ------ Go Test Runner ------
#
FROM godev AS tester

# run tests
RUN hack/test.sh

# upload coverage reports to Codecov.io: pass CODECOV_TOKEN as build-arg
ARG CODECOV_TOKEN
# default codecov bash uploader (sometimes it's worth to use GitHub version or custom one, to avoid bugs)
ARG CODECOV_BASH_URL=https://codecov.io/bash
# set Codecov expected env
ARG VCS_COMMIT_ID
ARG VCS_BRANCH_NAME
ARG VCS_SLUG
ARG CI_BUILD_URL
ARG CI_BUILD_ID
RUN if [ "$CODECOV_TOKEN" != "" ]; then curl -s $CODECOV_BASH_URL | bash -s; fi


#
# ------ Go Builder ------
#
FROM godev AS builder

# build binary
ARG VERSION
RUN hack/build.sh

#
# ------ Cronus CRON Event Provider image ------
#
FROM alpine:3.9

ENV GIN_MODE=release

RUN apk update && apk add --no-cache ca-certificates && apk upgrade

COPY --from=builder /go/src/github.com/codefresh-io/cronus/.bin/cronus /usr/local/bin/cronus

ENTRYPOINT ["/usr/local/bin/cronus"]
CMD ["server"]

ARG VCS_COMMIT_ID
LABEL org.label-schema.vcs-ref=$VCS_COMMIT_ID \
      org.label-schema.vcs-url="https://github.com/codefresh-io/cronus" \
      org.label-schema.description="cronus is a Cron Event Provider" \
      org.label-schema.vendor="Codefresh Inc." \
      org.label-schema.url="https://github.com/codefresh-io/cronus" \
      org.label-schema.docker.cmd="docker run -d --rm -p 80:8080 codefreshio/cronus server" \
      org.label-schema.docker.cmd.help="docker run -it --rm codefreshio/cronus --help"
