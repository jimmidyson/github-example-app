# GitHub Example App
[![CircleCI branch](https://img.shields.io/circleci/project/github/jimmidyson/github-example-app/master.svg)](https://circleci.com/gh/jimmidyson/github-example-app)
[![license](https://img.shields.io/github/license/jimmidyson/github-example-app.svg)](https://raw.githubusercontent.com/jimmidyson/github-example-app/master/LICENSE)
[![Docker Automated buil](https://img.shields.io/docker/automated/jimmidyson/github-example-app.svg)](https://hub.docker.com/r/rhipaas/github-example-app/)

**Pu**ll **Re**quest Bot enables automated pull request workflows, reacting to input from webhooks and
performing actions as configured.

Currently actions include:

* Labeling with `approved` label on pull request review approval.
* Automerging a PR once it has `approved` label and passes all required status checks.

## Running

```bash
$ github-example-app help run

Runs github-example-app.

Usage:
github-example-app run [flags]

Flags:
    --github-app-id int               GitHub app ID
    --github-app-private-key string   GitHub app private key file

Global Flags:
    --config string     config file (default is $HOME/.github-example-app.yaml)
    --log-level Level   log level (default info)
```

## Building

```bash
$ make
building: bin/amd64/github-example-app

$ make image
building: bin/amd64/github-example-app
Sending build context to Docker daemon 73.18 MB
Step 1/6 : FROM alpine:3.5
---> 88e169ea8f46
Step 2/6 : MAINTAINER Jimmi Dyson <jimmidyson@gmail.com>
---> Using cache
---> 3cd3ad11bf98
Step 3/6 : RUN apk update && apk upgrade && apk add ca-certificates && rm -rf /var/cache/apk
---> Using cache
---> ae9fde8c1cc7
Step 4/6 : ADD bin/amd64/github-example-app /github-example-app
---> 29cbebdf88fd
Removing intermediate container abec733e4481
Step 5/6 : USER 10000
---> Running in c61f53a8a9fe
---> 78549c7310e4
Removing intermediate container c61f53a8a9fe
Step 6/6 : ENTRYPOINT /github-example-app
---> Running in dcc313c83466
---> 9090fd17e37e
Removing intermediate container dcc313c83466
Successfully built 9090fd17e37e
image: rhipaas/github-example-app:3109e57-dirty
```