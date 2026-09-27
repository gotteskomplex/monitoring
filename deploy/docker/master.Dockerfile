# syntax=docker/dockerfile:1
# Build context: repository root.

FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
COPY api/ /src/api/
RUN npm run build

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist/ internal/master/webui/dist/
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w \
      -X github.com/gotteskomplex/monitoring/internal/platform/version.Version=${VERSION} \
      -X github.com/gotteskomplex/monitoring/internal/platform/version.Commit=${COMMIT} \
      -X github.com/gotteskomplex/monitoring/internal/platform/version.Date=${DATE}" \
    -o /out/master ./cmd/master

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/master /usr/local/bin/master
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/master"]
CMD ["serve"]
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s CMD ["/usr/local/bin/master", "healthcheck"]
