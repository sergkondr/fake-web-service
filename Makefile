APP_NAME := fakesvc
APP_VERSION := dev

build: test
	go build -ldflags="-X 'main.version=${APP_VERSION}'" -o ${APP_NAME} ./cmd/

test:
	go vet ./...
	go test -v ./...

docker:
	docker build  \
		--build-arg="APP_VERSION=${APP_VERSION}" \
		--build-arg="APP_NAME=${APP_NAME}" \
		-t sergkondr/${APP_NAME}:${APP_VERSION} .
	docker push sergkondr/${APP_NAME}:${APP_VERSION}
