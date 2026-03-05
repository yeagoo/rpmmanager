# Stage 1: Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.24-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN find internal/embed/dist -mindepth 1 ! -name '.gitkeep' -delete 2>/dev/null; cp -r web/dist/* internal/embed/dist/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o rpmmanager ./cmd/rpmmanager

# Stage 3: Runtime
FROM almalinux:9-minimal

RUN microdnf install -y \
    gnupg2 \
    createrepo_c \
    rpm-sign \
    rpmlint \
    tar \
    && microdnf clean all

# Install nfpm
ARG NFPM_VERSION=2.41.1
RUN rpm -i "https://github.com/goreleaser/nfpm/releases/download/v${NFPM_VERSION}/nfpm_${NFPM_VERSION}_amd64.rpm" || \
    echo "WARNING: nfpm installation failed, builds will not work"

COPY --from=backend /app/rpmmanager /usr/local/bin/rpmmanager

RUN useradd -r -m -d /data rpmmanager
RUN mkdir -p /data /etc/rpmmanager && chown -R rpmmanager:rpmmanager /data

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
