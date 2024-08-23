.DEFAULT_GOAL = test

APP_NAME := fakesvc
APP_VERSION := 0.1.0

test: lint
	go vet ./...
	go test -v ./... -count=1
.PHONY: test

lint:
	gofumpt -l -w .
	golangci-lint run --show-stats ./...
.PHONY: lint

build: test
	go build -ldflags="-X 'main.version=${APP_VERSION}'" -o ${APP_NAME} ./cmd/
.PHONY: build

docker:
	docker buildx build --push --no-cache --platform=linux/amd64,linux/arm64,linux/arm/v7 -t sergkondr/${APP_NAME}:${APP_VERSION} .
.PHONY: docker

deploy:
	 cat deployments/manifests/kubernetes-deploy.yaml | kapp deploy --namespace ${APP_NAME} --app ${APP_NAME} --diff-changes --yes --file -
.PHONY: deploy
