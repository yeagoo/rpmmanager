# RPM Manager

通用化的图形 RPM 包管理工具，支持多产品 RPM 构建、签名、发布和版本监控。

Go 后端 + React 前端 + SQLite，编译为单二进制文件，同时支持 Docker 部署。

## 功能概览

- **产品管理** — 定义多个产品（Caddy、LiteSpeed、PHP 等），配置上游源、目标发行版、打包参数
- **构建流水线** — 4 阶段自动化：下载 → 签名 → 发布 → 验证，实时 WebSocket 日志
- **GPG 密钥管理** — Web UI 导入/生成密钥，关联到产品进行 RPM 和 repomd 签名
- **仓库浏览** — 文件树浏览、.repo 文件预览、一键回滚
- **版本监控** — 自动检测 GitHub Release 新版本，可选自动触发构建
- **Webhook** — 支持 CI/CD 通过 API Token 触发构建

## 系统要求

### 宿主机部署

| 依赖 | 版本 | 说明 |
|------|------|------|
| Go | 1.24+ | 编译后端 |
| Node.js | 22+ | 编译前端 |
| createrepo_c | - | 生成 YUM/DNF 仓库元数据 |
| nfpm | 2.x | 构建 RPM 包 |
| gpg | 2.x | GPG 签名 |
| rpm / rpm-sign | - | RPM 签名 |

### Docker 部署

仅需 Docker 和 Docker Compose，所有依赖已包含在镜像中。

---

## 快速开始

### 方式一：Docker Compose（推荐）

```bash
# 克隆仓库
git clone https://github.com/ivmm/rpmmanager.git
cd rpmmanager

# 创建配置文件
cp config.example.yaml deploy/docker/config.yaml

# 编辑配置（至少修改 base_url）
vi deploy/docker/config.yaml

# 启动
cd deploy/docker
docker compose up -d
```

首次启动会在终端输出自动生成的管理员密码，查看日志获取：

```bash
docker compose logs rpmmanager | grep "Generated admin password"
```

访问 `http://localhost:8080` 登录。

### 方式二：预编译二进制

从 [GitHub Releases](https://github.com/ivmm/rpmmanager/releases) 下载对应架构的二进制文件：

```bash
# 下载（以 amd64 为例）
curl -Lo rpmmanager https://github.com/ivmm/rpmmanager/releases/latest/download/rpmmanager-linux-amd64
chmod +x rpmmanager

# 创建配置
mkdir -p /etc/rpmmanager
cp config.example.yaml /etc/rpmmanager/config.yaml
vi /etc/rpmmanager/config.yaml

# 运行
./rpmmanager serve --config /etc/rpmmanager/config.yaml
```

### 方式三：从源码编译

```bash
git clone https://github.com/ivmm/rpmmanager.git
cd rpmmanager

# 一键构建（前端 + 后端 → 单二进制）
make build

# 产物：./rpmmanager（约 13MB，内嵌前端资源）
./rpmmanager serve --config config.example.yaml
```

---

## 配置说明

配置文件为 YAML 格式，支持环境变量覆盖（前缀 `RPMMANAGER_`）。

```yaml
server:
  listen: "0.0.0.0:8080"       # 监听地址
  base_url: "https://rpms.example.com"  # 公开访问地址，用于 .repo 文件生成和 CORS

auth:
  username: "admin"             # 管理员用户名
  password_hash: ""             # bcrypt 哈希，留空则首次启动自动生成密码
  api_token: ""                 # Webhook/CI 用 API Token，留空则自动生成
  jwt_secret: ""                # JWT 签名密钥，留空则自动生成

database:
  path: "./data/rpmmanager.db"  # SQLite 数据库路径

storage:
  repo_root: "./data/repos"     # RPM 仓库存储根目录
  build_logs: "./data/logs"     # 构建日志目录
  temp_dir: "./data/tmp"        # 临时文件目录

gpg:
  home_dir: "./data/gnupg"      # GPG 密钥环目录（自动设置 0700 权限）

tools:
  nfpm_path: "nfpm"             # nfpm 可执行文件路径
  createrepo_path: "createrepo_c"
  gpg_path: "gpg"
  rpm_path: "rpm"
  rpmlint_path: "rpmlint"

monitor:
  enabled: true                 # 启用版本监控
  default_interval: "6h"        # 默认检查间隔
  github_token: ""              # GitHub API Token（提高速率限制）

log:
  level: "info"                 # 日志级别：debug/info/warn/error
  format: "text"                # 日志格式：text（彩色）/ json
```

### 环境变量覆盖

所有配置项均可通过环境变量设置，格式为 `RPMMANAGER_` + 大写路径（用 `_` 分隔）：

```bash
export RPMMANAGER_SERVER_LISTEN="0.0.0.0:9090"
export RPMMANAGER_DATABASE_PATH="/var/lib/rpmmanager/db.sqlite"
export RPMMANAGER_MONITOR_GITHUB_TOKEN="ghp_xxxx"
```

---

## 生产部署

### Systemd 服务

```bash
# 创建系统用户
sudo useradd -r -m -d /var/lib/rpmmanager -s /sbin/nologin rpmmanager

# 安装二进制
sudo cp rpmmanager /usr/local/bin/
sudo chmod +x /usr/local/bin/rpmmanager

# 创建目录
sudo mkdir -p /etc/rpmmanager /var/lib/rpmmanager /var/log/rpmmanager
sudo chown -R rpmmanager:rpmmanager /var/lib/rpmmanager /var/log/rpmmanager

# 配置文件
sudo cp config.example.yaml /etc/rpmmanager/config.yaml
sudo vi /etc/rpmmanager/config.yaml
# 修改 database.path, storage.* 路径指向 /var/lib/rpmmanager/

# 安装服务
sudo cp deploy/rpmmanager.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now rpmmanager

# 查看日志
sudo journalctl -u rpmmanager -f
```

配置文件中的路径建议修改为：

```yaml
database:
  path: "/var/lib/rpmmanager/rpmmanager.db"
storage:
  repo_root: "/var/lib/rpmmanager/repos"
  build_logs: "/var/log/rpmmanager"
  temp_dir: "/var/lib/rpmmanager/tmp"
gpg:
  home_dir: "/var/lib/rpmmanager/gnupg"
```

### 安装构建依赖

**RHEL / AlmaLinux / Rocky Linux 9：**

```bash
sudo dnf install -y gnupg2 createrepo_c rpm-sign rpmlint

# 安装 nfpm
sudo rpm -i https://github.com/goreleaser/nfpm/releases/download/v2.41.1/nfpm_2.41.1_amd64.rpm
```

**Fedora：**

```bash
sudo dnf install -y gnupg2 createrepo_c rpm-sign rpmlint
sudo rpm -i https://github.com/goreleaser/nfpm/releases/download/v2.41.1/nfpm_2.41.1_amd64.rpm
```

### Nginx 反向代理

```nginx
server {
    listen 443 ssl http2;
    server_name rpms.example.com;

    ssl_certificate     /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    client_max_body_size 100m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 支持（构建日志实时流）
    location ~ ^/api/builds/\d+/ws$ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 3600s;
    }
}
```

### Caddy 反向代理

```
rpms.example.com {
    reverse_proxy localhost:8080
}
```

Caddy 自动处理 WebSocket 升级，无需额外配置。

---

## Docker 部署详解

### docker-compose.yml

```yaml
services:
  rpmmanager:
    build:
      context: ../..
      dockerfile: deploy/docker/Dockerfile
    # 或使用预构建镜像：
    # image: ghcr.io/ivmm/rpmmanager:latest
    container_name: rpmmanager
    ports:
      - "8080:8080"
    volumes:
      - rpmmanager-data:/data
      - ./config.yaml:/etc/rpmmanager/config.yaml:ro
    environment:
      - RPMMANAGER_SERVER_LISTEN=0.0.0.0:8080
    restart: unless-stopped

volumes:
  rpmmanager-data:
```

### 常用操作

```bash
# 查看日志
docker compose logs -f rpmmanager

# 获取首次生成的密码
docker compose logs rpmmanager 2>&1 | grep "Generated admin password"

# 备份数据
docker compose exec rpmmanager cp /data/rpmmanager.db /data/rpmmanager.db.bak

# 更新版本
docker compose pull
docker compose up -d
```

### 数据持久化

容器内所有数据存储在 `/data` 目录下：

| 路径 | 说明 |
|------|------|
| `/data/rpmmanager.db` | SQLite 数据库 |
| `/data/repos/` | RPM 仓库文件 |
| `/data/logs/` | 构建日志 |
| `/data/tmp/` | 构建临时文件 |
| `/data/gnupg/` | GPG 密钥环 |

---

## 开发指南

### 开发环境

```bash
# 安装前端依赖
cd web && npm install && cd ..

# 同时启动前端和后端开发服务器
make dev
# 前端：http://localhost:5173（Vite HMR，代理 API 到后端）
# 后端：http://localhost:8080
```

### 构建命令

```bash
make build          # 完整构建（前端 + 后端 → 单二进制）
make build-backend  # 仅构建后端（跳过前端）
make test           # 运行 Go 测试
make clean          # 清理构建产物
```

### 项目结构

```
rpmmanager/
├── cmd/rpmmanager/         # CLI 入口
├── internal/
│   ├── auth/               # JWT + API Token 认证
│   ├── config/             # YAML 配置加载
│   ├── database/           # SQLite + 迁移
│   ├── distromap/          # 发行版映射 (EL8/EL9/Fedora/openEuler...)
│   ├── embed/              # go:embed 前端资源
│   ├── handler/            # HTTP 路由处理器
│   ├── models/             # 数据模型
│   ├── pipeline/           # 4 阶段构建流水线
│   ├── repository/         # 数据访问层
│   └── service/            # 业务逻辑
├── web/                    # React + TypeScript + shadcn/ui
├── deploy/
│   ├── docker/             # Dockerfile + docker-compose
│   └── rpmmanager.service  # systemd 单元文件
└── .github/workflows/      # CI/CD
```

---

## API 参考

所有 API 需要 JWT Token 或 API Token 认证（除 `/api/health` 和 `/api/auth/login`）。

### 认证

```bash
# JWT 登录
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your-password"}'

# 使用 JWT Token
curl -H "Authorization: Bearer <jwt-token>" http://localhost:8080/api/products

# 使用 API Token
curl -H "X-API-Token: <api-token>" http://localhost:8080/api/products
```

### 主要端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| POST | `/api/auth/login` | 登录 |
| GET | `/api/dashboard` | 仪表盘数据 |
| GET/POST | `/api/products` | 产品列表/创建 |
| GET/PUT/DELETE | `/api/products/:id` | 产品详情/更新/删除 |
| POST | `/api/products/:id/duplicate` | 复制产品 |
| GET/POST | `/api/builds` | 构建列表/触发 |
| GET | `/api/builds/:id` | 构建详情 |
| GET | `/api/builds/:id/log` | 构建日志 |
| POST | `/api/builds/:id/cancel` | 取消构建 |
| WS | `/api/builds/:id/ws` | 构建日志实时流 |
| GET | `/api/gpg-keys` | GPG 密钥列表 |
| POST | `/api/gpg-keys/import` | 导入密钥 |
| POST | `/api/gpg-keys/generate` | 生成密钥 |
| GET | `/api/gpg-keys/:id/export` | 导出公钥 |
| GET | `/api/repos` | 仓库列表 |
| GET | `/api/repos/:product/tree` | 仓库文件树 |
| POST | `/api/repos/:product/rollback` | 回滚仓库 |
| GET/PUT | `/api/monitors/:product_id` | 版本监控 |
| POST | `/api/webhook/:product` | Webhook 触发构建 |

### Webhook 示例

```bash
# 通过 Webhook 触发构建（使用 API Token）
curl -X POST https://rpms.example.com/api/webhook/caddy \
  -H "X-API-Token: your-api-token" \
  -H "Content-Type: application/json" \
  -d '{"version": "2.9.0"}'
```

---

## 常见问题

### 忘记管理员密码

删除配置文件中的 `password_hash` 字段（或设为空），重启服务会重新生成并打印新密码。

### GPG 签名失败

确认 GPG 密钥环目录权限正确：

```bash
ls -la /var/lib/rpmmanager/gnupg/
# 应为 drwx------ (700)
```

### createrepo_c 找不到

```bash
# RHEL/AlmaLinux/Rocky
sudo dnf install createrepo_c

# 或在配置文件中指定完整路径
tools:
  createrepo_path: "/usr/bin/createrepo_c"
```

### Docker 中构建 RPM 失败

确保容器有足够的内存（建议 >= 1GB）和磁盘空间。检查 nfpm 是否安装成功：

```bash
docker compose exec rpmmanager nfpm --version
```

---

## 许可证

MIT
