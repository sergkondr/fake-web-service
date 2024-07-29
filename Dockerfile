FROM golang:1.22.1 as builder
ARG APP_VERSION=dev
ARG APP_NAME=app

COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 go build -ldflags="-X 'main.version=${APP_VERSION}'" -o ${APP_NAME} /app/cmd/

FROM alpine:3.20
ARG APP_NAME=app
USER nobody
EXPOSE 8080

COPY --from=builder /app/${APP_NAME} /app/${APP_NAME}
ENTRYPOINT ["/app/${APP_NAME}"]
