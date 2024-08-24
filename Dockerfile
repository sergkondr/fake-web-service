FROM --platform=$BUILDPLATFORM golang:1.22.1 AS builder
ARG APP_VERSION=dev
ARG TARGETOS
ARG TARGETARCH

COPY . /app
WORKDIR /app
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 \
    go build -ldflags="-s -w -X 'main.version=${APP_VERSION}'" \
    -o fakesvc /app/cmd/

FROM gcr.io/distroless/static-debian12:nonroot

USER nobody
EXPOSE 8080

COPY --from=builder /app/fakesvc /app/fakesvc
ENTRYPOINT ["/app/fakesvc"]
