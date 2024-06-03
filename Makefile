build:
	sam build

deploy: build
	sam deploy \
		--template-file .aws-sam/build/template.yaml \
		--profile=grapple-sam-deployer \
		--stack-name grapple-dev \
		--capabilities CAPABILITY_IAM \
		--region us-west-1 --resolve-s3

run: up build
	sam local start-api --docker-network=backend_default --region local --env-vars=env.json

up:
	docker compose up --build -d

down:
	docker compose down