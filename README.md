# Duck2API

DuckDuckGo AI Chat 转 OpenAI 兼容 API 代理。支持 Chat Completions、图像生成/编辑、文件上传问答、语音转文字、文字转语音、推理模式、网络搜索等完整功能。
## 接口文档

curl 示例请查看：[API.md](API.md)

## 部署

### 编译部署

```bash
git clone https://github.com/aurora-develop/duck2api
cd duck2api
go build -o duck2api
chmod +x ./duck2api
./duck2api
```

### Docker 部署

```bash
docker run -d \
  --name duck2api \
  -p 8080:8080 \
  ghcr.io/aurora-develop/duck2api:latest
```

### Docker Compose 部署

```bash
mkdir duck2api && cd duck2api
wget https://raw.githubusercontent.com/aurora-develop/duck2api/main/docker-compose.yml
docker-compose up -d
```

## 功能概览

| 功能 | 端点 | 说明 |
|------|------|------|
| Chat Completions | `POST /v1/chat/completions` | 流式/非流式对话 |
| Responses API | `POST /v1/responses` | OpenAI Responses API |
| 图像生成 | `POST /v1/images/generations` | 文生图 |
| 图像编辑 | `POST /v1/images/edits` | 图生图/改图 |
| 文件上传 | `POST /v1/files` | 上传文件用于问答 |
| 文件管理 | `GET/DELETE /v1/files/:id` | 查询/删除文件 |
| 语音转文字 | `POST /v1/audio/transcriptions` | Whisper 兼容接口 |
| 文字转语音 | `POST /v1/audio/speech` | TTS 接口，支持 MP3/WAV/OGG |
| 模型列表 | `GET /v1/models` | 列出可用模型 |

## 快速开始

### 基础对话

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": true
  }'
```

### 推理模式 (Reasoning)

使用 `reasoning_effort` 参数控制推理深度：

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "messages": [{"role": "user", "content": "证明勾股定理"}],
    "reasoning_effort": "high"
  }'
```

支持的值：`none`（快速）、`low`、`medium`、`high`

### 网络搜索

设置 `web_search: true` 启用联网搜索：

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-nano",
    "messages": [{"role": "user", "content": "今天的科技新闻"}],
    "web_search": true
  }'
```

### 图像生成

```bash
curl http://localhost:8080/v1/images/generations \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "一只可爱的猫咪坐在窗台上",
    "model": "gpt-5.4-nano"
  }'
```

### 图像编辑

```bash
# JSON 方式 (base64)
curl http://localhost:8080/v1/images/edits \
  -H "Content-Type: application/json" \
  -d '{
    "image": "<base64编码的图片>",
    "prompt": "把猫改成蓝色"
  }'

# 文件上传方式
curl http://localhost:8080/v1/images/edits \
  -F "image=@cat.png" \
  -F "prompt=把猫改成蓝色"
```

### 文件上传与问答

```bash
# 1. 上传文件
curl http://localhost:8080/v1/files \
  -F "file=@document.pdf" \
  -F "purpose=assistants"

# 返回: {"id": "file-xxx", "object": "file", ...}

# 2. 使用文件进行问答（将文件内容作为上下文）
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-nano",
    "messages": [{"role": "user", "content": "请总结这个文档"}],
    "file_ids": ["file-xxx"]
  }'
```

### 语音转文字

```bash
curl http://localhost:8080/v1/audio/transcriptions \
  -F "file=@audio.webm" \
  -F "model=whisper-1"
```

支持格式：webm、ogg、mp3、wav、m4a、flac、opus、aac

### 文字转语音

```bash
curl http://localhost:8080/v1/audio/speech \
  -H "Content-Type: application/json" \
  -d '{"model":"tts-1","input":"你好世界","voice":"alloy","response_format":"mp3"}' \
  --output speech.mp3
```

支持格式：mp3、wav、ogg、flac、aac（底层使用 Duck.ai 的 WebRTC + OpenAI Realtime API）

## 支持的模型

| 模型 | 类型 | 说明 |
|------|------|------|
| `gpt-5.4-nano` | 推理 | OpenAI 最新轻量推理模型 |
| `gpt-5.4-mini` | 推理 | OpenAI 推理模型 |
| `claude-haiku-4-5` | 通用 | Anthropic Claude |
| `tinfoil/gpt-oss-120b` | 通用 | OpenAI GPT-1.5 120B |
| `mistral-small` | 通用 | Mistral AI |



### Remote MCP tools (Responses API)

Remote MCP is available through `POST /v1/responses`. The gateway currently supports MCP Streamable HTTP transport and discovers tools through the official Go MCP SDK. It runs the model-led tool loop: the model chooses whether to call a tool, which permitted tool to call, and whether more research is needed. There is no fixed search sequence. `MCP_MAX_TOOL_CALLS` is an operational safety ceiling, not a planned search count.

The upstream Duck.ai adapter currently represents system instructions as ordinary conversation text rather than a privileged system channel. Consequently, local registry validation, the tool allow-list, and approval settings are the actual execution boundary; never register write-capable or destructive tools for unattended research.

The gateway does not accept arbitrary MCP destinations from a request. Register trusted endpoints in the server environment using `MCP_SERVERS`. The request's `server_label` and `server_url` must match a registered entry, and the effective tool set is the intersection of the registry allow-list and request `allowed_tools`.

Example environment value (replace the placeholder with your own trusted endpoint):

```json
[
  {
    "server_label": "research",
    "server_url": "https://YOUR-TRUSTED-MCP-HOST/mcp",
    "allowed_tools": ["search", "fetch"],
    "require_approval": "never"
  }
]
```

Keep authentication headers in the server-side registry, never in client requests or source control:

```json
{
  "server_label": "research",
  "server_url": "https://YOUR-TRUSTED-MCP-HOST/mcp",
  "allowed_tools": ["search", "fetch"],
  "require_approval": "never",
  "headers": {
    "Authorization": "Bearer YOUR_TOKEN"
  }
}
```

The corresponding Responses request uses the same registered endpoint and an explicit approval policy:

```json
{
  "model": "gpt-6-luna",
  "tools": [
    {
      "type": "mcp",
      "server_label": "research",
      "server_url": "https://YOUR-TRUSTED-MCP-HOST/mcp",
      "allowed_tools": ["search", "fetch"],
      "require_approval": "never"
    }
  ],
  "input": "Research the requested question and cite the sources."
}
```

**Execution safety:** tool execution is blocked unless `require_approval` is `never` in both the trusted server registry and the request. This first implementation does not yet implement the interactive approval-response round trip; therefore, use automatic execution only for explicitly allow-listed, read-only tools on servers you trust. Omitted or non-string approval policies are treated as approval-required. Registry policy cannot be weakened by the request.

MCP endpoints must use HTTPS by default. `MCP_ALLOW_HTTP=true` is intended only for a trusted local deployment. The gateway rejects user-supplied endpoints that do not match the registry, restricts redirect behavior through the SDK transport, rejects reserved header overrides, and bounds call time and returned payload size.

Supported environment variables:

| Variable | Default | Purpose |
|---|---:|---|
| `MCP_SERVERS` | empty | Trusted MCP server registry (JSON array) |
| `MCP_ALLOW_HTTP` | `false` | Permit HTTP only for trusted local deployments |
| `MCP_MAX_TOOL_CALLS` | `64` | Maximum tool calls per response |
| `MCP_TOOL_TIMEOUT_SECONDS` | `120` | Timeout for each MCP call |
| `MCP_MAX_RESULT_BYTES` | `65536` | Maximum serialized tool-result size |

## 高级设置

### 环境变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `Authorization` | API 认证 Key | `Bearer your_key` |
| `PROXY_URL` | 代理地址 | `http://proxy:8080` |
| `PREFIX` | URL 前缀 | `/api` |
| `TLS_CERT` | TLS 证书路径 | `/path/to/cert.pem` |
| `TLS_KEY` | TLS 密钥路径 | `/path/to/key.pem` |

### 代理池

支持 `proxies.txt` 文件配置多个代理（每行一个）：

```
http://proxy1:8080
http://proxy2:8080
socks5://proxy3:1080
```

## 鸣谢

感谢各位大佬的 PR 支持。

## 参考项目

- https://github.com/xqdoo00o/ChatGPT-to-API

## License

MIT License
