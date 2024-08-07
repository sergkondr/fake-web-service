.DEFAULT_GOAL = test

APP_NAME := fakesvc
APP_VERSION := dev

test:
	go vet ./...
	go test -v ./...
.PHONY: test

lint:
	golangci-lint run --show-stats ./...
.PHONY: lint

build: test
	go build -ldflags="-X 'main.version=${APP_VERSION}'" -o ${APP_NAME} ./cmd/
.PHONY: build

docker:
	docker build --build-arg="APP_VERSION=${APP_VERSION}" -t sergkondr/${APP_NAME}:${APP_VERSION} .
	docker push sergkondr/${APP_NAME}:${APP_VERSION}
.PHONY: docker

deploy:
	 cat deployments/manifests/kubernetes-deploy.yaml | kapp deploy --namespace ${APP_NAME} --app ${APP_NAME} --diff-changes --file -
.PHONY: deploy
