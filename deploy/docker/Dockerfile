# Stage 1: Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.24-alpine AS backend

ARG VERSION=dev
ARG COMMIT=unknown
ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN rm -rf internal/embed/dist && cp -r web/dist internal/embed/dist
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o rpmmanager ./cmd/rpmmanager

# Stage 3: Runtime
FROM almalinux:9-minimal

ARG TARGETARCH=amd64

RUN microdnf install -y \
    gnupg2 \
    createrepo_c \
    rpm-sign \
    rpmlint \
    tar \
    && microdnf clean all

# Install nfpm (supports x86_64/aarch64)
ARG NFPM_VERSION=2.45.0
RUN rpm -i "https://github.com/goreleaser/nfpm/releases/download/v${NFPM_VERSION}/nfpm-${NFPM_VERSION}-1.$(uname -m).rpm"

COPY --from=backend /app/rpmmanager /usr/local/bin/rpmmanager

RUN useradd -r -m -d /data rpmmanager
RUN mkdir -p /data/repos /etc/rpmmanager && chown -R rpmmanager:rpmmanager /data && chmod 755 /data /data/repos

VOLUME ["/data", "/etc/rpmmanager"]
EXPOSE 8080
USER rpmmanager

ENV RPMMANAGER_DATABASE_PATH=/data/rpmmanager.db
ENV RPMMANAGER_STORAGE_REPO_ROOT=/data/repos
ENV RPMMANAGER_STORAGE_BUILD_LOGS=/data/logs
ENV RPMMANAGER_STORAGE_TEMP_DIR=/data/tmp
ENV RPMMANAGER_GPG_HOME_DIR=/data/gnupg

ENTRYPOINT ["rpmmanager"]
CMD ["serve", "--config", "/etc/rpmmanager/config.yaml"]
