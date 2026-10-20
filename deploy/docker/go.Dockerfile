# Builds crosswatc-api or crosswatc-worker: docker build --build-arg SERVICE=api -f deploy/docker/go.Dockerfile .
FROM golang:1.26 AS build
ARG SERVICE
ARG VERSION=dev
ARG COMMIT=none
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY db ./db
COPY internal ./internal
COPY services ./services
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOMAXPROCS=2 go build -p 1 -trimpath \
    -ldflags "-s -w -X github.com/agnaldom/crosswatc/internal/version.Version=${VERSION} -X github.com/agnaldom/crosswatc/internal/version.Commit=${COMMIT}" \
    -o /out/app ./services/${SERVICE}

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]
