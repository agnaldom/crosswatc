# Builds crosswatc-api or crosswatc-worker: docker build --build-arg SERVICE=api -f deploy/docker/go.Dockerfile .
FROM golang:1.25 AS build
ARG SERVICE
ARG VERSION=dev
ARG COMMIT=none
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY db ./db
COPY internal ./internal
COPY services ./services
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/agnaldom/crosswatc/internal/version.Version=${VERSION} -X github.com/agnaldom/crosswatc/internal/version.Commit=${COMMIT}" \
    -o /out/app ./services/${SERVICE}

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]
