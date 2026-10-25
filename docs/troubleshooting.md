# 故障排查

先确认失败发生在哪一层：HTTP 服务、设备连接、麦克风／播放、外部 AI。基础演示步骤见 [演示指南](demo.md)，启动命令见 [贡献指南](../CONTRIBUTING.md)。

## 快速定位

以下命令对应贡献指南的本机 18080 端口；使用默认 Compose 时将端口改为 8080。

```bash
curl --fail --silent --show-error http://localhost:18080/healthz
curl --fail --silent --show-error http://localhost:18080/api/groups
```

健康检查返回 `ok` 只证明服务可达。浏览器控制台的 `/favicon.ico` 404 不影响对讲；优先检查 API、WebSocket 和音频相关错误。

| 现象 | 检查与处理 | 恢复标准 |
|---|---|---|
| Go 工具链／依赖下载失败 | 检查 `go version`、`go env GOTOOLCHAIN GOPROXY` 和当前网络；模块要求见 `go.mod`。先完成 `go mod download`，不要随意降低 `go.mod` 版本 | 依赖下载和 `make test` 成功 |
| `make test` 提示 race／CGO 不可用 | 检查 `go env CGO_ENABLED`，在支持的平台安装 C 编译工具链并启用 CGO。无 race 的测试不能代替 CI 的竞态检查 | `go test -race ./...` 成功 |
| 启动报端口占用 | 换一个 `PTT_ADDR`，同时更新 `PTT_BASE_URL` 和浏览器访问端口；不要停止不属于本项目的服务 | 新端口健康检查成功 |
| 页面能打开，但麦克风不可用 | 检查是否为 localhost 或可信 HTTPS；检查站点权限及系统麦克风权限；使用支持 WebCodecs Opus 的浏览器 | 开机日志显示麦克风就绪，回声测试能复读 |
| 已连接但听不到声音 | 两台设备确认同一房间；取消静音／只听队长；检查系统输出设备、浏览器静音及耳机；先单独测试回声机器人 | 听到回声，再完成双设备对讲 |
| 两个标签页互相断线 | 查看设备 ID；同一个设备 ID 的新连接会替换旧连接。使用 `/device?device=demo-a` 和 `/device?device=demo-b` | 两个不同 ID 同时在线 |
| 一直提示抢麦失败或只听不讲 | 查看当前发言者和群模板。导游模板只有队长可讲；普通成员不能抢占已有发言 | 普通群空闲时能发言，或正确设置队长角色 |
| 回放没有内容 | 录音至少三帧；确认发言已结束且在当前群。设备回放排除自己的发言，让另一台设备讲话再试 | 管理页看到录音，接收设备可以回放 |
| 扫码后找不到服务器 | 检查二维码里的地址与 `PTT_BASE_URL`；手机上的 localhost 不代表你的电脑 | 扫码设备能访问分享链接；通话另需 HTTPS |
| 改了 JS／CSS，页面没有变化 | 静态文件通过 `go:embed` 打包；重新启动 Go 服务或重新构建 Docker 镜像，再刷新页面 | 浏览器资源内容包含新改动 |
| 管理操作返回 403 | 配置 `PTT_AUTH_TOKEN` 后，管理页用 `/admin?token=你的令牌` 打开一次；接口用 Bearer 令牌。令牌变更后重新打开带新令牌的管理页 | 解散、设队长或导入等操作通过鉴权 |
| 管理页看着在线，操作却失败 | 当前管理页刷新失败会保留旧数据。检查健康接口和浏览器 Network；恢复服务后刷新，不要只依据旧在线数判断 | API 和页面状态恢复更新 |
| 没有转写或关键词预警 | 未配置 AI Key 时依赖浏览器转写；浏览器可能不支持或识别失败。配置 Key 后检查服务端 ASR 日志和录音 URL；没有文字就不能触发关键词匹配 | 录音元数据出现 transcript，包含预警词时收到通知 |
| 摘要只是“谁说了几条” | 没有可用转写时就是该降级结果；有文字但 LLM 不可用时返回转写列表 | 按实际配置验证降级，或修复转写和 LLM 后得到摘要 |
| Docker 一启动就退出／反复重启 | 先读 `docker compose logs --tail=50 ptt`，特别检查 `/data/recordings` 权限和 AI 配置 | 日志显示 listening，健康接口返回 ok |

## Docker 数据卷权限

仓库镜像使用 nonroot 用户。若日志出现：

```text
open archive: mkdir /data/recordings: permission denied
```

新命名卷可按照 [首次启动步骤](demo.md#docker首次启动与数据卷) 为卷根目录设置 UID/GID `65532:65532`。使用 bind mount 或已有数据时，先确认实际挂载位置、文件所有权并备份，再修正服务需要写入的目录权限。不要为解决问题直接删除数据卷或使用 `chmod 777`。

镜像是 distroless，不包含普通 shell；`docker exec ... sh` 失败不代表服务异常。通过容器日志、宿主机的 HTTP 请求和 Docker 挂载信息排查。

## ASR 与 Kodo

1. 在管理页找到一条完整发言，确认可以播放本地录音。
2. 查看服务日志是否有 `asr`、`summary llm failed` 或 Kodo 上传错误；日志中的录音 URL 不要直接公开，它可能包含临时签名。
3. 默认 ASR 模式需要远端服务下载录音，`PTT_BASE_URL=http://localhost:...` 只适用于本机访问；使用可达地址或已上传 Kodo 的预签名地址。
4. Kodo 上传失败时检查 Bucket、区域、Endpoint 和凭据；仍可先用本地回放验证录音。
5. 使用 Compose 时确认变量已显式传入容器。详见 [配置传递说明](demo.md#配置怎样进入容器)。

规则助手、浏览器语音识别、服务端 ASR 和 LLM 是不同环节。文字指令能成功，不意味着浏览器语音识别或 ASR 正常。

## 报告问题时附什么

提供服务版本／提交、启动方式、浏览器版本、最小复现步骤、预期与实际结果、脱敏后的相关日志。对讲问题补充两个设备 ID、房间号以及是否能通过回声测试；AI 问题补充采用的 ASR 模式和是否存在转写。

不要附真实令牌、API Key、AK／SK、带签名的链接或未经允许公开的录音。能提供合成数据和最小复现时优先使用它们。
