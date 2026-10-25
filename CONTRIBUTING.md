# 贡献指南

欢迎提交代码、测试和文档。第一次贡献建议从一个可复现的问题开始，把复现步骤、改动和验证结果放在同一个 PR 中。

## 开发环境

- Go 1.25.1 或更高的兼容版本（以 [go.mod](go.mod) 为准）。首次运行可能自动下载工具链和依赖。
- Git；使用 Makefile 时需要 `make`。也可以执行其中对应的 Go 命令。
- `make test` 使用竞态检测，需要支持 race detector 的平台、启用 CGO，并安装 C 编译工具链。
- 浏览器演示使用支持 WebCodecs Opus 的 Chrome / Edge；需要麦克风及其权限。
- Docker 和 Compose 是可选部署方式，本地开发不依赖它们。浏览器端没有 npm 安装或打包步骤。

在你自己的 fork 或工作副本根目录执行：

```bash
go version
go mod download
make test
make vet
make build
```

预期：测试与静态检查通过，`bin/` 下生成 `ptt-server` 和 `ptt-sim`。CI 还检查 Go 格式和 Docker 构建，见 [.github/workflows/ci.yml](.github/workflows/ci.yml)。没有外部 AI Key 也能运行这些 Go 测试；AI 客户端测试使用本地模拟服务。

## 启动一个独立演示实例

在项目根目录执行以下命令，避免覆盖已有 `./data` 中的演示状态：

```bash
PTT_ADDR=127.0.0.1:18080 \
PTT_BASE_URL=http://localhost:18080 \
PTT_DATA_DIR=./data/contributor-demo \
PTT_AI_KEY= PTT_MCP_ENDPOINT= PTT_AUTH_TOKEN= \
make demo
```

打开 `http://localhost:18080/device` 和 `http://localhost:18080/admin`。另开终端执行：

```bash
curl --fail --silent --show-error http://localhost:18080/healthz
curl --fail --silent --show-error http://localhost:18080/api/groups
```

预期：健康检查返回 `ok`；群列表包含“回声测试”，服务日志出现回声机器人上线。健康接口只说明 HTTP 服务可达，不能证明麦克风、音频播放或外部 AI 正常。

停止时在服务终端按 Ctrl+C。再次使用同一个数据目录会保留群组和录音；需要全新演示时换一个未使用过的 `PTT_DATA_DIR`。详细操作见 [演示与部署指南](docs/demo.md)。

`web/static` 通过 `go:embed` 编入服务：修改 HTML、JS 或 CSS 后，需要停止并重新运行 `make demo`，再刷新浏览器。仅刷新网页不会让旧进程加载磁盘上的新文件。

## 从哪里读代码

| 修改内容 | 入口 |
|---|---|
| 启动、环境变量、演示数据 | `cmd/ptt-server/`、`internal/config/` |
| 群组和设备规则、持久化 | `internal/store/` |
| 管理与设备 REST 接口 | `internal/api/` |
| 抢麦、实时音频转发、设备事件 | `internal/voice/`，配合 [协议文档](docs/protocol.md) |
| 录音、转写、摘要 | `internal/archive/`、`internal/ai/` |
| MCP 与内置文字／语音助手 | `internal/mcp/`、`internal/assistant/` |
| 管理页、模拟器、扫码页 | `web/static/admin.*`、`device.*`、`join.*`；音频处理在 `audio.js` |
| 设备联调和压测 | `cmd/ptt-sim/`、`internal/sim/`，配合 [固件指南](docs/firmware.md) |

## 修改和验证

1. 建立独立分支，一次解决一个问题。前端体验修改尽量复用现有接口；协议或响应结构调整时同步对应文档。
2. 修复后端缺陷时，先加能够在原实现上失败的回归用例，覆盖实际触发条件。测试数据用临时目录，外部服务用模拟响应。
3. 先运行相关包测试，例如摘要修改执行 `go test -race ./internal/ai`，再执行 `make test`、`make vet`、`make build`。
4. 对修改过的 Go 文件执行 `gofmt -w <文件路径>`；用 `gofmt -l .` 检查，预期没有输出。最后运行 `git diff --check`。
5. 前端修改在浏览器验证正常操作、空数据、请求失败和重复点击。涉及对讲时再验证双设备、抢麦冲突和断线恢复。Go 测试不能替代真实浏览器的麦克风验证。
6. 文档中的命令在独立数据目录或测试卷验证。需要硬件、证书、外部账号的步骤，请在 PR 中说明实际验证到了哪里。

不需要为了修改页面引入前端框架。也不要把真实 API Key、管理令牌、录音或本地数据加入提交。

## PR 描述建议

说明这些信息即可，篇幅与改动大小匹配：

- **问题**：什么操作会遇到什么结果？最好给出最小复现。
- **修改后**：用户能观察到什么变化？接口或数据格式是否变化？
- **验证**：实际执行过的命令、浏览器操作及结果。页面修改可附前后截图。
- **范围**：尚未验证的外部服务或硬件，以及已知限制。

提交前查看 `git diff` 和 `git status`，确保 PR 只包含本次改动。文档链接以仓库内相对路径为主，方便 fork 和分支预览。
