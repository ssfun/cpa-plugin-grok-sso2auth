# Grok SSO → Auth — CLIProxyAPI 插件

将 **xAI / Grok SSO Cookie** 经 OAuth Device Flow 转换成 CLIProxyAPI 可用的 `type=xai` / `auth_kind=oauth` 凭证，并通过宿主 `host.auth.save` **直接导入** auth-dir。

**版本** `v0.3.0` ｜ **平台** Linux / macOS / Windows / FreeBSD ｜ **License** MIT

参考：

- [CLIProxyAPI 插件开发](https://help.router-for.me/cn/plugin/development.html)
- [cpa-plugin-gemini-cli](https://github.com/router-for-me/cpa-plugin-gemini-cli)（构建 / Release 流水线）
- [cpa-plugin-grok-panel](https://github.com/TizenryA/cpa-plugin-grok-panel)（管理 UI + host.auth 模式）

---

## 功能

| 能力 | 说明 |
|------|------|
| **管理 UI** | CPA 管理中心菜单「Grok SSO 导入」；自动复用管理中心认证，无需重复输入 Management Key |
| **SSO → xai JSON** | Device Flow：`device/code` → `verify` → `approve` → `token` → `userinfo` |
| **一键导入** | 转换成功后调用 `host.auth.save` 写入 auth-dir（文件名 `xai-{email}.json`） |
| **批量** | 多行 SSO / `email----password----sso`，限流时自动提高账号间隔，成功后缓慢回落 |
| **可靠性** | 默认验证 SSO；阶段重试、账号级限流重跑、重新申请并打开 Device Code |
| **CLI 标志** | `--grok-sso-cookie` / `--grok-sso-file` 等 |

---

## 仓库结构

```text
cpa-plugin-grok-sso2auth/
├── cmd/grok-sso2auth/     # 插件入口（C ABI + 业务）
│   ├── abi.go             # cliproxy_plugin_init / call / free / shutdown
│   ├── handlers.go        # plugin.register / management.* / command_line.*
│   ├── sso.go             # SSO Device Flow → XAIAuthFile
│   ├── ui.go              # 管理中心 HTML 页面
│   ├── fileio.go
│   └── *_test.go
├── .github/
│   ├── workflows/build.yml
│   └── scripts/package-release.go
├── dist/                  # make build / make package 输出目录（gitignore）
├── registry.json          # 插件商店 registry 片段
├── Makefile
├── go.mod
├── LICENSE
└── README.md
```

构建产物命名遵循 CPA 插件商店约定：

```text
grok-sso2auth_<version>_<goos>_<goarch>.zip   # zip 根目录直接是 grok-sso2auth.{so|dylib|dll}
checksums.txt                                 # sha256 汇总
```

---

## 快速开始

### 1. 本地编译

需要 Go 1.26+，并启用 CGO：

```bash
# 当前平台动态库 → dist/grok-sso2auth.{dylib|so|dll}
make build

# 打 zip + sha256
make package VERSION=0.3.0

# 安装到默认插件目录
make install
# 或: make install PLUGINS_DIR=/path/to/plugins
```

### 2. 配置 CLIProxyAPI

```yaml
plugins:
  enabled: true
  dir: "plugins"   # 或绝对路径，例如 ~/.cli-proxy-api/plugins
  configs:
    grok-sso2auth:
      enabled: true
      priority: 1
```

重启或热加载后，管理中心应出现菜单 **「Grok SSO 导入」**，资源页：

```text
/v0/resource/plugins/grok-sso2auth/status
```

也可用管理 API 确认：

```bash
curl -H "Authorization: Bearer <management-key>" \
  http://127.0.0.1:8317/v0/management/plugins
```

### 3. 使用管理 UI

1. 登录管理中心时勾选 **记住密码**
2. 打开管理中心 → **Grok SSO 导入**，页面会自动复用当前管理认证
3. 粘贴 SSO Cookie（或 `email----sso` 多行）
4. 点击 **开始转换并导入**
5. 在结果表中查看每个账号的文件名、尝试次数和限流状态

自动认证依赖插件资源页与管理中心同源，并从管理中心的 `cli-proxy-auth` 持久会话读取认证。若未勾选“记住密码”，请回到管理中心重新登录；插件页面不会再次索要或保存 Management Key。

页面 API 全部走：

```text
POST /v0/management/plugins/grok-sso2auth/convert
POST /v0/management/plugins/grok-sso2auth/convert-import
```

### 4. 命令行标志

由 CPA 宿主加载插件后暴露：

```bash
./cli-proxy-api \
  --grok-sso-cookie 'eyJ...' \
  --grok-sso-email 'user@example.com'

./cli-proxy-api \
  --grok-sso-file ./sso_list.txt \
  --grok-sso-delay 45 \
  --grok-sso-max-delay 180 \
  --grok-sso-retries 8 \
  --grok-sso-account-retries 3
```

| 标志 | 说明 |
|------|------|
| `--grok-sso-cookie` | 单个 SSO JWT |
| `--grok-sso-file` | 列表文件（一行一个 JWT，或 `email----sso`） |
| `--grok-sso-email` | 可选 email 覆盖 |
| `--grok-sso-delay` | 自适应批量账号基础间隔秒，默认 45 |
| `--grok-sso-max-delay` | 自适应间隔上限秒，默认 180 |
| `--grok-sso-retries` | Device / Verify / Approve 阶段最大重试次数，默认 8 |
| `--grok-sso-account-retries` | 限流时整个账号的最大尝试次数，默认 3 |
| `--grok-sso-validate` | 是否先访问 accounts.x.ai 校验 SSO，默认开启 |

---

## 输出凭证格式

与 CLIProxyAPI 内置 xAI OAuth（`internal/auth/xai.TokenStorage`）一致：

```json
{
  "type": "xai",
  "auth_kind": "oauth",
  "access_token": "...",
  "refresh_token": "...",
  "token_type": "Bearer",
  "expires_in": 21600,
  "expired": "2026-08-06T12:00:00Z",
  "last_refresh": "2026-08-06T06:00:00Z",
  "email": "user@example.com",
  "sub": "...",
  "base_url": "https://api.x.ai/v1",
  "token_endpoint": "https://auth.x.ai/oauth2/token",
  "redirect_uri": "http://127.0.0.1:56121/callback",
  "disabled": false,
  "id_token": "..."
}
```

文件名：`xai-{email}.json`（无 email 时用 `xai-{sub}.json`）。

---

## Release / 插件商店

推送 `v*` tag 后，GitHub Actions 会：

1. `go test` / `go vet`
2. 多平台 `c-shared` 构建（linux/darwin/windows/freebsd）
3. 打包 `grok-sso2auth_<ver>_<os>_<arch>.zip` + 单文件 sha256
4. 汇总 `checksums.txt` 并创建/更新 GitHub Release

本地模拟：

```bash
make package VERSION=0.3.0
# 产物在 dist/
```

`registry.json` 可供自建插件商店源引用：

```yaml
plugins:
  store-sources:
    - "https://raw.githubusercontent.com/ssfun/cpa-plugin-grok-sso2auth/main/registry.json"
```

---

## 安全说明

- 插件与宿主**同进程**运行，仅应安装可信来源的构建产物。
- 管理 UI 的 resource 页本身**不鉴权**；转换请求走 `/v0/management/...`，仍由宿主校验 Management Key。
- 页面只读取管理中心已经持久化的认证并放入请求头，不提供再次输入 Management Key 的入口，也不会把它写入插件配置。
- 管理中心未勾选“记住密码”或资源页跨源部署时，页面无法取得只存在父页面内存中的认证，需要回管理中心重新登录并持久化会话。
- “转换并导入”响应不返回 OAuth JSON 或物理路径；仅 `/convert` API 会按调用方明确请求返回转换后的 JSON。
- 不要把 SSO、access/refresh token 打进日志或 resource HTML。
- 本仓库源码与 CI **不包含**任何真实密钥。

---

## 开发

```bash
go test ./...
go vet ./...
make build
```

核心转换逻辑在 `cmd/grok-sso2auth/sso.go`。Device / Verify / Approve 分阶段重试；Verify 或 Approve 限流后会重新申请并再次打开 Device Code。批量间隔默认 45 秒，限流时按 `max(current×1.8, current+25, 45)` 提升（最高 180 秒），成功后按 `current×0.92` 缓慢回落，并加入 0–10 秒抖动。

---

## License

[MIT](LICENSE)
