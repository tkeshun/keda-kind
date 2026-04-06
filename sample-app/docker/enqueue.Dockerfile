# syntax=docker/dockerfile:1

FROM golang:1.25 AS build
WORKDIR /src

RUN mkdir -p /out

RUN --mount=type=bind,source=go.mod,target=/src/go.mod,readonly \
    --mount=type=bind,source=go.sum,target=/src/go.sum,readonly \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN --mount=type=bind,source=.,target=/workspace,readonly \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /workspace && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/enqueue ./sample-app/cmd/enqueue

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/enqueue /usr/local/bin/enqueue
ENTRYPOINT ["/usr/local/bin/enqueue"]
