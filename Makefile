.DEFAULT_GOAL = test

APP_NAME := fakesvc
APP_VERSION := dev

.PHONY: fmt lint test docker deploy

fmt:
	gofumpt -w .

lint:
	@test -z "$$(gofumpt -l .)"
	golangci-lint run --show-stats ./...

test:
	go vet ./...
	go test -v ./... -count=1

build:
	go build -ldflags="-X 'main.version=${APP_VERSION}'" -o ${APP_NAME} ./cmd/

docker:
	docker buildx build --push --no-cache --platform=linux/amd64,linux/arm64,linux/arm/v7 -t sergkondr/${APP_NAME}:${APP_VERSION} .

deploy:
	 cat deployments/manifests/kubernetes-deploy.yaml | kapp deploy --namespace ${APP_NAME} --app ${APP_NAME} --diff-changes --yes --file -
