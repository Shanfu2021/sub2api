# CLAUDE.md

本文件记录这个 fork 的实际运维方式、发布约定和高频坑点，供后续协作时直接复用。

## 1. 项目与生产环境

- 仓库工作目录：`/root/shushanfu/sub2api`
- 当前常用工作分支：`publish/fork-main-sync`
- 用户生产分支：`shanfu-prod`
- fork 远端：`fork = https://github.com/Shanfu2021/sub2api.git`
- upstream 远端：`origin = https://github.com/Wei-Shaw/sub2api.git`

- 生产域名：`https://api.ise.it.com`
- systemd 服务：`sub2api.service`
- 线上二进制路径：`/opt/sub2api/sub2api`
- 线上数据目录：`/opt/sub2api/data`
- 线上配置文件：`/opt/sub2api/config.yaml`

- 数据库：
  - host: `127.0.0.1`
  - port: `5432`
  - user: `sub2api`
  - dbname: `sub2api`
  - 实际密码以 `/opt/sub2api/config.yaml` 为准

## 2. 站点定制约定

- 站点名：`天才程序员拼车站`
- 副标题：`一群程序员共建的高质量 AI API 拼车站`
- 客服 QQ：`1198716953`
- 站点 logo 静态文件：`/logo.png`
- 文档页静态入口：`/docs/`
- 商品总入口：`https://pay.ldxp.cn/shop/CN8U85FN`

## 3. 发布原则

这台服务器之前多次因为本地前端打包导致 CPU 100% 卡死，因此遵守以下规则：

- 不在生产机本地跑完整前端构建。
- 前端改动通过 GitHub Actions `Build Self-Hosted Binary` 构建。
- 构建产物下载到本机后，只替换二进制并重启服务。
- 尽量使用轻量验证，不做高负载全量构建。

## 4. GitHub 凭据与环境变量

发布脚本依赖以下环境变量：

```bash
export GITHUB_OWNER="Shanfu2021"
export GITHUB_REPO="sub2api"
export GITHUB_TOKEN="你的 PAT"
```

如需部署到当前机器，默认值已内置：

```bash
export SERVICE_NAME="sub2api.service"
export INSTALL_PATH="/opt/sub2api/sub2api"
export INSTALL_USER="sub2api"
export INSTALL_GROUP="sub2api"
```

## 5. 轻量发布 SOP

### 5.1 推送当前 HEAD 到生产分支

```bash
cd /root/shushanfu/sub2api
tools/release_helper.sh push shanfu-prod
```

### 5.2 触发 GitHub Actions 构建

```bash
cd /root/shushanfu/sub2api
tools/release_helper.sh dispatch shanfu-prod
```

### 5.3 查询该分支最近一次成功构建的 run id

```bash
cd /root/shushanfu/sub2api
tools/release_helper.sh latest-run shanfu-prod
```

如果要锁定某一次新提交，避免误拿上一条成功构建，可传 `head sha`：

```bash
cd /root/shushanfu/sub2api
tools/release_helper.sh latest-run shanfu-prod <commit-sha>
```

### 5.4 等待指定 run 完成

```bash
cd /root/shushanfu/sub2api
tools/release_helper.sh wait-run <run-id>
```

### 5.5 下载指定 run 的产物

```bash
cd /root/shushanfu/sub2api
tools/release_helper.sh download <run-id>
```

下载后目录默认在：

```bash
/root/shushanfu/sub2api/.tmp/release-artifacts/run-<run-id>/
```

下载脚本会自动把二进制整理到根目录，优先使用：

```bash
/root/shushanfu/sub2api/.tmp/release-artifacts/run-<run-id>/sub2api
```

如果 GitHub artifact 仍保留原始目录结构，则也可能存在：

```bash
/root/shushanfu/sub2api/.tmp/release-artifacts/run-<run-id>/dist/sub2api
```

### 5.6 部署二进制

```bash
cd /root/shushanfu/sub2api
tools/release_helper.sh deploy /root/shushanfu/sub2api/.tmp/release-artifacts/run-<run-id>/sub2api
```

`deploy` 现在也支持直接传下载目录：

```bash
cd /root/shushanfu/sub2api
tools/release_helper.sh deploy /root/shushanfu/sub2api/.tmp/release-artifacts/run-<run-id>/
```

### 5.7 一键部署某分支最近一次成功构建

```bash
cd /root/shushanfu/sub2api
tools/deploy_latest.sh shanfu-prod
```

如果要只部署某一个 commit 对应的成功构建：

```bash
cd /root/shushanfu/sub2api
tools/deploy_latest.sh shanfu-prod <commit-sha>
```

说明：

- `deploy_latest.sh` 只负责“取最近一次成功完成的 Build Self-Hosted Binary 构建并部署”。
- 它不会自动 push，也不会自动 dispatch。
- 如果不传 `commit-sha`，它会取该分支最近一次成功构建。
- 所以完整动作通常仍是：`push` -> `dispatch` -> `wait-run` -> `deploy_latest.sh`。
- 如果要确保部署的是刚推上去的新提交，推荐显式传 `commit-sha`。

## 6. 文档与前端入口约定

- 自定义文档静态页放在：
  - `frontend/public/docs/index.html`
  - `frontend/public/docs.html`
- 当前前端已改为：
  - 若后台 `doc_url` 为空，自动回落到 `/docs/`
  - 这样首页、头部、用量页都会显示文档入口

## 7. 已知坑点与修复经验

### 7.1 首页首屏配置注入容易漏字段

风险：

- `dto.PublicSettings` 新增字段后，SSR 注入结构 `PublicSettingsInjectionPayload` 可能漏掉。
- 即使结构体定义了字段，也可能忘记在 `GetPublicSettingsForInjection` 里赋值。

当前已做：

- `backend/internal/handler/dto/public_settings_injection_schema_test.go`
  - 防止“字段漏定义”
- `backend/internal/service/setting_service_public_test.go`
  - 防止“字段定义了但没赋值”，已覆盖 `contact_qr_code_url` 和 `doc_url`

后续规则：

- 任何新增公共设置，如果首页首屏依赖它，就必须同时检查：
  - `dto.PublicSettings`
  - `service.PublicSettingsInjectionPayload`
  - `GetPublicSettingsForInjection`
  - 对应测试

### 7.2 生产机不要本地跑前端 build

- 之前 `pnpm build` 导致 2C2G 服务器卡死。
- 所有包含前端产物的发布，走 GitHub Actions。

### 7.3 站点静态资源改动也需要重新打包嵌入二进制

例如：

- `frontend/public/logo.png`
- `frontend/public/docs/*`

这些虽然是“静态文件”，但当前是嵌入到 Go 二进制里，不是热更新目录。
改完后仍需要重新构建二进制并部署。

### 7.4 OpenAI APIKey 兼容上游默认开启透传

现象：

- 如果 OpenAI 账号类型是 `apikey`，并且 `credentials.base_url` 指向第三方兼容上游或另一个 sub2api，中间多一层网关。
- 不开启 `openai_passthrough` 时，请求会走标准 OpenAI 改写链路：解析大 JSON、模型/字段/fast policy/图片/previous_response_id 处理后再转发。
- 对 Codex 长上下文请求，这会放大首 token 延迟，尤其是 `cache_read_tokens > 100k`、`reasoning.effort=high/xhigh` 的请求。

规则：

- 新增 OpenAI APIKey 兼容上游时，如果配置了自定义 `base_url`，默认在 `accounts.extra` 设置：

```json
{
  "openai_passthrough": true,
  "openai_apikey_responses_websockets_v2_enabled": false,
  "openai_apikey_responses_websockets_v2_mode": "off"
}
```

- `openai_passthrough=true` 表示本机只替换鉴权并尽量保持原始 body 转发，减少本机解析/重写开销。
- WebSocket v2 只在明确确认该上游支持时再开启；对第三方兼容上游默认保持关闭，避免协议不兼容。
- 官方 OpenAI 直连 APIKey（没有自定义 `base_url`）不强制套用这条规则。
- 标准 HTTP 上游 Transport 已显式启用 HTTP/2；TLS 指纹伪装路径仍保持 HTTP/1.1，避免破坏伪装行为。
- 排查首 token 慢时，先查服务日志 `OpenAI passthrough slow first token`。重点字段：
  - `upstream_header_ms`：本机发到上游并收到响应头的耗时。
  - `first_output_after_headers_ms`：收到响应头后，到首个可输出 SSE 事件的耗时。
  - `upstream_body_bytes`：本机发给上游的请求体大小。

应急 SQL：

```sql
UPDATE accounts
SET extra = jsonb_set(COALESCE(extra, '{}'::jsonb), '{openai_passthrough}', 'true'::jsonb, true),
    updated_at = NOW()
WHERE platform = 'openai'
  AND type = 'apikey'
  AND COALESCE(credentials->>'base_url', '') <> '';
```

直接改库后，要写入调度 outbox 或重启服务，确保账号快照刷新：

```sql
INSERT INTO scheduler_outbox (event_type, account_id)
SELECT 'account_changed', id
FROM accounts
WHERE platform = 'openai'
  AND type = 'apikey'
  AND COALESCE(credentials->>'base_url', '') <> '';
```

## 8. 当前新增工具

- `tools/release_helper.sh`
  - `push`
  - `dispatch`
  - `latest-run`
  - `wait-run`
  - `download`
  - `deploy`
- `tools/deploy_latest.sh`
  - 拉取指定分支最新成功构建并部署
- `tools/render_docs.mjs`
  - 临时文档渲染工具：把 `/opt/sub2api/data/docs-source.md` 渲染成 `/opt/sub2api/data/public/docs/index.html`
  - 当前不是正式发布链的一部分
  - 线上主文档仍以 `frontend/public/docs/index.html` 为准

## 9. 建议的日常工作流

```bash
cd /root/shushanfu/sub2api

# 1. 本地改代码
# 2. 轻量测试 / 检查差异
# 3. 提交
git add ...
git commit -m "..."

# 4. 推送到生产分支
tools/release_helper.sh push shanfu-prod

# 5. 触发构建并拿到 run id
RUN_ID=$(tools/release_helper.sh dispatch shanfu-prod)

# 6. 等 GitHub Actions 成功
tools/release_helper.sh wait-run "$RUN_ID"

# 7. 部署当前 HEAD 对应构建
HEAD_SHA=$(git rev-parse HEAD)
tools/deploy_latest.sh shanfu-prod "$HEAD_SHA"
```

## 10. 本次相关修改

本次为了减少后续反复排查，已经做了以下固化：

- 修复首页注入漏掉 `contact_qr_code_url` 的 bug
- 补 `GetPublicSettingsForInjection` 回归测试
- 文档链接默认回落到 `/docs/`
- 新增 GitHub 发布与部署辅助脚本
- 将流程和约定写入本文件
