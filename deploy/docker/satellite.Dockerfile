# syntax=docker/dockerfile:1
# Build context: repository root.
# Note: ICMP checks (phase 2) need the NET_RAW capability (docker run --cap-add NET_RAW).

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w \
      -X github.com/gotteskomplex/monitoring/internal/platform/version.Version=${VERSION} \
      -X github.com/gotteskomplex/monitoring/internal/platform/version.Commit=${COMMIT} \
      -X github.com/gotteskomplex/monitoring/internal/platform/version.Date=${DATE}" \
    -o /out/satellite ./cmd/satellite

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/satellite /usr/local/bin/satellite
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/satellite"]
CMD ["version"]
