# 演示与部署指南

本指南用于第一次跑通项目及 hackathon 展示。开发环境和检查命令见 [贡献指南](../CONTRIBUTING.md)，遇到问题见 [故障排查](troubleshooting.md)。

## 先选择演示范围

| 能力 | 所需条件 | 未配置或不可用时 |
|---|---|---|
| 建群、入群、切群、管理页 | 本地服务和浏览器 | 不需要 AI Key |
| 实时对讲、回声测试、录音回放 | 麦克风、安全上下文、WebCodecs Opus | 可以演示控制操作，不能据此证明语音链路可用 |
| 输入文字让助手建群／切群 | 内置规则解析器 | 不需要 LLM；可用固定话术演示 |
| 对助手说话、客户端转写 | 浏览器语音识别能力及其运行条件 | 改用文字输入；不能保证离线可用 |
| 服务端 ASR | `PTT_AI_KEY`；默认 URL 模式还需录音 URL 可被服务端 ASR 访问 | 未配置时接受客户端转写；配置后 ASR 失败不等于会自动采用客户端转写 |
| LLM 摘要／助手 | AI 配置与可用的外部服务 | 助手退回规则解析；摘要退回转写内容，再退回发言人统计 |
| Kodo 录音镜像 | Bucket、AK、SK、区域／Endpoint 配置 | 录音仍可保存在本地；不配置 Kodo 不影响基础对讲 |

先用无需外部 AI 的流程确认基础功能，再加入转写和摘要。具体变量以 [README 配置表](../README.md#配置) 为准。

## 本机演示：三分钟

按照 [独立实例启动步骤](../CONTRIBUTING.md#启动一个独立演示实例) 启动，以下地址均使用 `http://localhost:18080`。

1. **单设备试音**：打开 `/device?device=demo-a`，开机并允许麦克风，点击加入“回声测试”，按住 PTT 说一句话再松开。预期先显示发射状态，随后机器人复读。失败先排查，不要直接进入双设备演示。
2. **建群**：在 `/admin` 新建“演示车队”，记录本次生成的房间号。房间号不是固定常量，不要照抄截图里的号码。
3. **双设备入群**：再打开 `/device?device=demo-b`，两台设备分别输入同一个房间号。显式使用不同的 `device` 参数，避免复制标签页时继承相同身份。
4. **对讲和抢麦**：A 按住 PTT，B 收听；A 不松开时，B 也按 PTT，预期 B 收到抢麦失败提示。两台设备均使用普通成员身份。推荐戴耳机，减少同机双标签的声音回授。
5. **回放**：让 A 说一段超过一秒的话，松开后在 B 点击回放，或在管理页“发言记录”播放。设备回放会排除自己的发言；短于三帧的误触不入存档。

验收记录：记录房间号、两个设备 ID、能否收听、抢麦结果和回放结果。自动化接口检查不能替代这一轮听音确认。

## 加两分钟：控制与摘要

- 在 A 的助手中输入“建个群叫演示老乡群”，再输入“下一个频道”，最后输入“切到演示车队”回到双设备所在群。预期当前频道和管理页成员状态随之变化；规则助手无需 AI Key。
- B 产生几条发言后，A 询问“我错过了什么”。有转写且 LLM 可用时生成摘要；否则显示转写或发言人统计。这些都是预期降级，不要把统计结果介绍为 LLM 摘要。
- 若已配置并确认转写链路可用，再让 B 说“前面封路，走辅路”，观察转写和预警。没有转写时不能只凭音频期待关键词预警。

现场外部服务不可用时，演示基础对讲、回放、管理页和文字控制即可。说明当前使用的能力及降级结果。

## Docker：首次启动与数据卷

当前镜像以 UID/GID `65532:65532`（nonroot）运行。新建命名卷的根目录可能为 root 所有，导致首次启动报 `mkdir /data/recordings: permission denied`。以下步骤为**独立演示项目**初始化一个新卷；不针对已有生产数据卷，也不需要将服务长期改为 root。

在仓库根目录执行：

```bash
docker compose -p ptt-walkie-demo build
docker volume create ptt-walkie-demo_ptt-data
docker run --rm -v ptt-walkie-demo_ptt-data:/data alpine:3 chown 65532:65532 /data
docker compose -p ptt-walkie-demo up -d
docker compose -p ptt-walkie-demo logs --tail=30 ptt
curl --fail --silent --show-error http://localhost:8080/healthz
```

预期：日志显示服务监听和回声机器人上线，健康检查返回 `ok`。Compose 默认开启演示模式；页面为 `http://localhost:8080/device`、`/admin`。依赖镜像和依赖包首次下载需要网络。

停止演示但保留数据卷：

```bash
docker compose -p ptt-walkie-demo down
```

这里没有 `-v`，下次启动会继续使用原数据。临时验证想换端口时，可使用下面的单次容器方式（不要同时运行同一项目的常驻服务）：

```bash
docker compose -p ptt-walkie-demo run --rm \
  --publish 127.0.0.1:18086:8080 \
  -e PTT_BASE_URL=http://localhost:18086 ptt
```

### 配置怎样进入容器

当前 [docker-compose.yml](../docker-compose.yml) 只显式传入 `PTT_BASE_URL`、`PTT_AI_KEY`、`PTT_AI_MODEL`、`PTT_AUTH_TOKEN`、`PTT_MCP_ENDPOINT`、`PTT_DEMO`。宿主机的其他变量或 `.env` 文件内容不会因此全部自动进入容器。

例如，仅验证 ASR 模式配置传递（不填真实 Key 时不会调用 ASR）：

```bash
docker compose -p ptt-walkie-demo run --rm \
  --publish 127.0.0.1:18086:8080 \
  -e PTT_BASE_URL=http://localhost:18086 \
  -e PTT_ASR_MODE=openai ptt
```

真实接入时，还需通过 `-e PTT_AI_KEY` 等参数传入已在宿主机设置的变量，或在自己的 Compose override 中增加对应 `environment`；`PTT_KODO_*` 同理。不要把真实凭据写进提交。普通 `go run`／`make demo` 读取进程环境，不会自动加载 `.env`。

## 局域网与外部 AI

- 手机或另一台电脑访问时，把 `PTT_BASE_URL` 设为它们可以访问的服务地址，并确保监听地址允许访问。二维码中的 `localhost` 会指向扫码设备自身。
- 麦克风要求 HTTPS 或 localhost 安全上下文。局域网 HTTP 地址能打开页面，不代表能通话。可使用 HTTPS 反向代理，或配置服务端 `PTT_TLS_CERT` / `PTT_TLS_KEY`；证书文件在容器中还需要挂载，配置值填写容器内路径。
- 默认 `PTT_ASR_MODE=qiniu` 让 ASR 服务拉取录音 URL：仅本机可访问的 `localhost` URL 无法供远端服务使用。可使用公网可达的 `PTT_BASE_URL`，或正确配置 Kodo，让 ASR 使用预签名 URL。
- `PTT_ASR_MODE=openai` 直接上传音频，但仍需要目标服务支持对应接口，并配置正确的 Key、Base URL 和模型。

外部 AI、Kodo、TLS 和实体设备需要对应环境单独验收。本指南中的本地健康检查及 Docker 启动成功不代表这些集成已经通过。
