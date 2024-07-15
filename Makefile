ENV?=test

build:
	sam build --config-env=${ENV}

deploy: build
	echo "Deploying SAM template to ${ENV} test environment"
	sam deploy \
		--profile=grapple-sam-deployer \
		--config-env=${ENV} \
		--config-file=$$PWD/samconfig.yaml

run: up build
	sam local start-api --docker-network=backend_default --region local --config-env=local

up:
	docker compose up --build -d

down:
	docker compose down