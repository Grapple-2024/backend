# README

## Pre-requisites

To get started developing in this project, complete the pre-requisites below.

1. [Install Docker and Docker Compose](https://docs.docker.com/compose/install/)
2. Install Golang 1.22.0 - Recommended to install via [GVM (Go Version Manager)](https://github.com/moovweb/gvm)
3. Reach out to Jordan to get the samconfig.yml file. The SAM config file contains all of the necessary SAM template parameter overrides to deploy to local, test, and prod environments.

## Local Development
First, make sure you have a valid `samconfig.yml` file in the root directory of this repository.

To start the backend and all necessary dependencies on your local machine:
```sh
make run
```


This make recipe will first start the dependencies (mongodb) defined in `compose.yaml` via Docker Compose. It will then start a local instance of the API Gateway utilizing `sam local start-api`.


## Deploying to the Cloud

To deploy the backend (Lambda Function, Roles, API Gateway, and other resources):

```sh
make deploy ENV={ENV}
```

> **Note:** `{ENV}` can either be `test` or `prod`. Be cautious when deploying to the `prod`` environment -- triple check all changes are tested in `test` before proceeding.