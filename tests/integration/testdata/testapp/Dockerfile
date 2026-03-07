# Stage 1: Cross-compile the test binary
FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X main.version=0.1.0" -o testapp-amd64 .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-X main.version=0.1.0" -o testapp-arm64 .

# Stage 2: Serve via nginx
FROM nginx:1-alpine
RUN mkdir -p /usr/share/nginx/html/download/testapp/0.1.0/linux/amd64 \
             /usr/share/nginx/html/download/testapp/0.1.0/linux/arm64
COPY --from=builder /src/testapp-amd64 /usr/share/nginx/html/download/testapp/0.1.0/linux/amd64/testapp
COPY --from=builder /src/testapp-arm64 /usr/share/nginx/html/download/testapp/0.1.0/linux/arm64/testapp
