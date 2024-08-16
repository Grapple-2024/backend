ENV?=test

build:
	sam build --config-env=${ENV}

deploy: build
	@echo "This will build AND deploy the current source code to Grapple's ${ENV} environment. Do you want to proceed? (Y/n)"
	@read choice; if [ $$choice != "Y" ]; then echo aborting; exit 1; fi
	@echo "Proceeding with deployment..."; \
	sam deploy \
		--profile=grapple-sam-deployer \
		--config-env=${ENV} \
		--config-file=$$PWD/samconfig.yml

### LOCAL DEVELOPMENT RECIPES
# Runs the post-signup lambda (cmd/post-signup-lambda)
EVENT?=./cmd/post-signup-lambda/testdata/event.json
run-post-signup: up build
	sam local invoke --docker-network=backend_default CreateProfileOnSignupLambda --event ${EVENT}

# Runs the grapple backend lambda (cmd/backend) LOCAL ONLY
PORT?=8080
run: up build
	sam local start-api --port=${PORT} --docker-network=backend_default --region us-west-1 --config-env=local --config-file=$$PWD/samconfig.yml

up:
	docker compose up --build -d

down:
	docker compose down