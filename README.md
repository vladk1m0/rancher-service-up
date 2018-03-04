# Rancher service upgrade tool for GitLab.  

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)[![Build Status](https://api.travis-ci.org/vladk1m0/rancher-service-up.svg?branch=master)](https://travis-ci.org/vladk1m0/rancher-service-up)[![Docker Pulls](https://img.shields.io/docker/pulls/vladk1m0/docker-rancher-service-up.svg)](https://hub.docker.com/r/vladk1m0/docker-rancher-service-up/)

This project inspired by [https://github.com/cdrx/rancher-gitlab-deploy](https://github.com/cdrx/rancher-gitlab-deploy) and it will upgrade existing services as part of your CI workflow.

**rancher-service-up** is a tool for deploying containers built with GitLab CI onto your Rancher infrastructure.

It fits neatly into the `gitlab-ci.yml` workflow and requires minimal configuration. 

Both GitLab's built in Docker registry and external Docker registries are supported.

`rancher-service-up` will pick as much of its configuration up as possible from environment variables set by the GitLab CI runner.

This tool is not suitable if your services are not already created in Rancher. It will upgrade existing services, but will not create new ones.  
If you need to create services you should use `rancher-compose` in your CI workflow, but that means storing any secret environment variables in your Git repo.

## Installation

I recommend you use the pre-built container:

https://hub.docker.com/r/vladk1m0/docker-rancher-service-up

But you can install the command locally, with `go`, if you prefer:

```
go get github.com/vladk1m0/rancher-service-up
```

## Usage

You will need to create a set of API keys in Rancher and save them as secret variables in GitLab for your project.

Three secret variables are required:

`RANCHER_URL` (eg `https://rancher.example.com`)

`RANCHER_ACCESS_KEY`

`RANCHER_SECRET_KEY`

Rancher supports two kind of API keys: environment and account. You can use either with this tool, but if your account key has access to more than one environment you'll need to specify the name of the environment with the --env flag. This is so that the tool can upgrade find the service in the right place. For example, in your `gitlab-ci.yml`:

```
deploy:
  stage: deploy
  image: vladk1m0/docker-rancher-service-up
  script:
    - rancher-service-up --env=development
```

`rancher-service-up` will use the GitLab group and project name as the stack and service name by default. For example, the project:

`http://gitlab.example.com/frontend/website`

will upgrade the service called `website` in the stack called `frontend`.

If the names of your services don't match your repos in GitLab 1:1, you can change the service that gets upgraded with the `--stack` and `--service` flags:

```
deploy:
  stage: deploy
  image: vladk1m0/docker-rancher-service-up
  script:
    - rancher-service-up --stack frontend --service website
```

You can change the image (or :tag) used to deploy the upgraded containers with the `--image` option:

```
deploy:
  stage: deploy
  image: vladk1m0/docker-rancher-service-up
  script:
    - rancher-service-up --image registry.example.com/frontend/website:1.2
```

You may use this with the `$CI_BUILD_TAG` environment variable that GitLab sets.

`rancher-service-up`'s default upgrade strategy is to upgrade containers one at time, waiting 2s between each one. It will start new containers after shutting down existing ones, to avoid issues with multiple containers trying to bind to the same port on a host. It will wait for the upgrade to complete in Rancher, then mark it as finished. The upgrade strategy can be adjusted with the flags in `--help` (see below).

## GitLab CI Example

Complete gitlab-ci.yml:

```
image: docker:latest
services:
  - docker:dind

stages:
  - build
  - deploy

build:
  stage: build
  script:
    - docker login -u gitlab-ci-token -p $CI_BUILD_TOKEN registry.example.com
    - docker build -t registry.example.com/group/project .
    - docker push registry.example.com/group/project

deploy:
  stage: deploy
  image: vladk1m0/docker-rancher-service-up
  script:
    - rancher-service-up
```

A more complex example:

```
deploy:
  stage: deploy
  image: vladk1m0/docker-rancher-service-up
  script:
    - rancher-service-up --env production --stack frontend --service website --image site:1.2
```

## Help

```
$ rancher-service-up --help

Rancher service upgrade tool version: 1.0.0
Usage: rancher-service-up [OPTIONS]
  -batch-interval int        Number of seconds to wait between upgrade batches. (default 2)
  -batch-size int
        Number of containers to upgrade at once. (default 1)  
  -debug
        Enable debug mode.
  -env string
        The name of the environment in Rancher. (default "Default")
  -finish-upgrade
        Mark the upgrade as finished after it completes. (default true)
  -image string
        The new image[:tag] of the service in Rancher to upgrade. (Required!)
  -key string
        The environment or account API key. (Required!) (default "RANCHER_ACCESS_KEY")
  -new-sidekick-image value
        If specified, replace the sidekick image[:tag] with this one during the upgrade.
  -secret string
        The secret for the access API key. (Required!) (default "RANCHER_SECRET_KEY")
  -service string
        The name of the service in Rancher to upgrade. (Required!) (default "CI_PROJECT_NAME")
  -stack string
        The name of the stack in Rancher. (Required!) (default "CI_PROJECT_NAMESPACE")
  -start-first
        Should Rancher start new containers before stopping the old ones.
  -upgrade-sidekicks
        Upgrade service sidekicks at the same time.
  -upgrade-timeout int
        How long to wait, in seconds, for the upgrade to finish before exiting. (default 180)
  -url string
        The URL for your Rancher server, eg: http://rancher:8080. (Required!) (default "RANCHER_URL")
  -version
        Display app version.
  -wait-for-upgrade-to-finish
        Wait for Rancher to finish the upgrade before this tool exits. (default true)
  --help 
        Show this message and exit.
```

## License

MIT
