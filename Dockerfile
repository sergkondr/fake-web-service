FROM golang:1.22.1 as builder
ARG APP_VERSION=dev

COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 go build -ldflags="-X 'main.version=${APP_VERSION}'" -o fakesvc /app/cmd/

FROM alpine:3.20

USER nobody
EXPOSE 8080

COPY --from=builder /app/fakesvc /app/fakesvc
ENTRYPOINT ["/app/fakesvc"]
