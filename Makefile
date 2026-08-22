.DEFAULT_GOAL = test

APP_NAME := fakesvc
APP_VERSION := dev
IMAGE ?= sergkondr/$(APP_NAME):$(APP_VERSION)
PLATFORMS ?= linux/amd64,linux/arm64

.PHONY: fmt lint test test-race build docker docker-multiarch deploy

fmt:
	gofumpt -w .

lint:
	@test -z "$$(gofumpt -l .)"
	golangci-lint run --show-stats ./...

test:
	go vet ./...
	go test -v ./... -count=1

test-race:
	go test -race ./... -count=1

build:
	go build -ldflags="-X 'main.version=${APP_VERSION}'" -o bin/${APP_NAME} ./cmd/

docker:
	docker build --build-arg APP_VERSION=$(APP_VERSION) -t $(IMAGE) .

docker-multiarch:
	docker buildx build --push --platform=$(PLATFORMS) --build-arg APP_VERSION=$(APP_VERSION) -t $(IMAGE) .

deploy:
	 cat deployments/manifests/kubernetes-deploy.yaml | kapp deploy --namespace ${APP_NAME} --app ${APP_NAME} --diff-changes --yes --file -
