# Stage 1 - build the React/Vite frontend
FROM node:22.22-alpine3.23 AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ .
ARG VERSION=dev
RUN sed -i "s/\"version\": *\"[^\"]*\"/\"version\": \"${VERSION}\"/" package.json && \
    npm run build

# Stage 2 - compile the Go binary
FROM golang:1.25-alpine3.23 AS go-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
ARG VERSION=dev
ARG BUILD_TIME
ARG GO_VERSION
RUN BUILD_TIME=${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)} && \
    GO_VERSION=${GO_VERSION:-$(go version | awk '{print $3}')} && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath \
      -ldflags "-X github.com/tlmanz/allure-hub/internal/transport/handler.Version=${VERSION} \
                -X 'github.com/tlmanz/allure-hub/internal/transport/handler.BuildTime=${BUILD_TIME}' \
                -X 'github.com/tlmanz/allure-hub/internal/transport/handler.GoVersion=${GO_VERSION}'" \
      -o /allure-hub ./cmd/server

# Stage 3 - final runtime image
# Node is required to run the Allure 3 CLI (npm package: allure).
FROM node:22.22-alpine3.23
RUN npm install -g allure@3.3.1 \
 && addgroup -S app && adduser -S app -G app

COPY --from=go-builder  /allure-hub        /usr/local/bin/allure-hub
COPY --from=frontend-builder /app/dist     /app/web

VOLUME ["/data"]
EXPOSE 8080

USER app
CMD ["/usr/local/bin/allure-hub"]
