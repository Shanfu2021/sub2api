# HZZ 分支构建与部署

本文档记录如何使用 `hzz` 分支构建并部署当前这套企业/代理扩展版本。生产部署目录按当前机器约定为 `/root/sub2api-deploy`，应用源码目录为 `/root/sub2api`。

## 目标状态

- 源码分支：`hzz`
- 应用镜像：`ghcr.io/h-2szz/sub2api:hzz`
- Compose 服务：`sub2api`
- 数据库服务：`sub2api-postgres`
- Redis 服务：`sub2api-redis`
- 对外端口：`0.0.0.0:8080`
- 数据目录：`/root/sub2api-deploy/data`

Postgres 和 Redis 不需要因为前端/后端代码更新而重建。正常只替换 `sub2api` 应用容器。

## 1. 确认源码分支

```bash
cd /root/sub2api
git fetch origin hzz
git checkout hzz
git pull --ff-only origin hzz
git status --short --branch
```

期望看到当前分支是 `hzz`，并且工作区干净：

```text
## hzz...origin/hzz
```

## 2. 优先使用 GitHub 远程构建

仓库的 `.github/workflows/hzz-image.yml` 会在推送 `hzz` 分支时构建并推送：

- `ghcr.io/h-2szz/sub2api:hzz`
- `ghcr.io/h-2szz/sub2api:hzz-<完整提交SHA>`

提交并推送：

```bash
cd /root/sub2api
git push origin hzz
```

等待远程镜像出现：

```bash
SHA="$(git rev-parse HEAD)"
docker manifest inspect "ghcr.io/h-2szz/sub2api:hzz-${SHA}"
```

如果返回 manifest JSON，说明远程构建已经完成。然后拉取并部署：

```bash
docker pull ghcr.io/h-2szz/sub2api:hzz
cd /root/sub2api-deploy
docker compose up -d --no-deps --force-recreate sub2api
```

## 3. 远程构建不可用时本地构建

如果 GitHub Actions 排队失败、GHCR 标签迟迟不存在，或者当前机器无法拉取最新远程镜像，可以本地低负载构建同样的镜像。

```bash
cd /root/sub2api
SHA="$(git rev-parse HEAD)"
SHORT_SHA="$(git rev-parse --short HEAD)"

DOCKER_BUILDKIT=0 docker build \
  --network=host \
  --build-arg VERSION=hzz \
  --build-arg COMMIT="${SHA}" \
  -f deploy/Dockerfile \
  -t "ghcr.io/h-2szz/sub2api:hzz-local-${SHORT_SHA}" \
  -t ghcr.io/h-2szz/sub2api:hzz \
  .
```

部署本地构建出的 `hzz` 标签：

```bash
cd /root/sub2api-deploy
docker compose up -d --no-deps --force-recreate sub2api
```

本机硬件较弱时，不要并行跑大规模测试和 Docker 构建。构建完成后可以清理悬空镜像：

```bash
docker image prune -f
```

## 4. 验证部署

检查容器状态：

```bash
docker ps --format '{{.Names}} {{.Image}} {{.ID}} {{.Status}} {{.Ports}}' | grep sub2api
```

健康检查：

```bash
curl -fsS http://127.0.0.1:8080/health
```

期望输出：

```json
{"status":"ok"}
```

确认运行镜像：

```bash
docker inspect sub2api --format '{{.Image}} {{.Config.Image}} {{.State.Health.Status}}'
```

查看启动日志：

```bash
docker logs --tail 80 sub2api
```

## 5. 常见问题

### 远程镜像标签不存在

运行：

```bash
SHA="$(cd /root/sub2api && git rev-parse HEAD)"
docker manifest inspect "ghcr.io/h-2szz/sub2api:hzz-${SHA}"
```

如果输出 `manifest unknown`，说明 GitHub Actions 还没推送这次提交对应的镜像。可以继续等待，也可以按第 3 节本地构建。

### 页面没变化

确认应用容器已经重建：

```bash
cd /root/sub2api-deploy
docker compose up -d --no-deps --force-recreate sub2api
```

然后强制刷新浏览器，或清理浏览器缓存。

### 磁盘空间紧张

先查看空间：

```bash
df -h /
docker system df
```

清理悬空镜像：

```bash
docker image prune -f
```

不要删除 `/root/sub2api-deploy/postgres_data`、`/root/sub2api-deploy/redis_data`、`/root/sub2api-deploy/data`，这些是运行数据。
