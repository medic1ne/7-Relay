# 7-Relay

**Smart AI Router — Single binary, 40+ providers, auto-fallback, token saver.**

> Go rewrite of [9Router](https://github.com/decolua/9router). Lighter, faster, more secure.

```
┌─────────────┐
│  Your CLI   │  Claude Code, Codex, Cursor, Cline...
└──────┬──────┘
       │ http://localhost:20128/v1
       ↓
┌─────────────────────────────────────┐
│           7-RELAY                   │
│  • RTK Token Saver (20-40% saved)  │
│  • Format Translation              │
│  • Smart 3-Tier Fallback           │
│  • E2E Encrypted Key Vault         │
│  • Built-in MITM Proxy             │
└──────┬──────────────────────────────┘
       │
       ├─→ Tier 1: Subscription (Claude/GPT)
       ├─→ Tier 2: Cheap (GLM/MiniMax)
       └─→ Tier 3: Free (Kiro/Vertex)
```

## Quick Start

```bash
# Install
go install github.com/medic1ne/7-Relay/cmd/7relay@latest

# Or download binary
curl -sL https://github.com/medic1ne/7-Relay/releases/latest/download/7relay-linux-amd64 -o 7relay
chmod +x 7relay && sudo mv 7relay /usr/local/bin/

# Run
7relay
```

Dashboard: http://localhost:20128
API: http://localhost:20128/v1

## Features

| Feature | Status | Description |
|---|---|---|
| Core Router | ✅ | OpenAI-compatible API proxy |
| 3-Tier Fallback | ✅ | Subscription → Cheap → Free |
| Multi-Account | ✅ | Round-robin load balancing |
| RTK Token Saver | ✅ | Compress tool output 20-40% |
| Caveman Mode | ✅ | Terse output, fewer tokens |
| Ponytail Mode | ✅ | YAGNI-first code generation |
| MITM Proxy | 🚧 | HTTP/HTTPS/WebSocket intercept |
| E2E Key Vault | 🚧 | ChaCha20 encrypted API keys |
| TUI Dashboard | 🚧 | Terminal-based management UI |
| Local Inference | 🚧 | Ollama/llama.cpp integration |
| Plugin System | 🚧 | WASM-based extensibility |
| Web3 Auth | 🚧 | Wallet-based authentication |

## Providers

### Free
- **Kiro AI** — Claude 4.5 + GLM-5 (50 credits/month)
- **OpenCode Free** — No auth required
- **Vertex AI** — $300 GCP credits

### Paid (40+)
OpenAI, Anthropic, Gemini, DeepSeek, Groq, xAI, Mistral, Perplexity, Together, Fireworks, Cerebras, Cohere, NVIDIA, SiliconFlow, OpenRouter, GLM, Kimi, MiniMax, and more.

## CLI Tools Supported

Claude Code, Codex, Cursor, Cline, Copilot, OpenClaw, OpenCode, Antigravity, Continue, Roo, Kilo Code, Devin CLI, Grok Build, and any OpenAI-compatible client.

## Configuration

Config file: `~/.7relay/config.json`

```json
{
  "port": 20128,
  "providers": [
    {
      "name": "openai",
      "type": "openai",
      "base_url": "https://api.openai.com/v1",
      "api_key": "sk-...",
      "models": ["gpt-4o", "gpt-4o-mini"],
      "tier": 2,
      "enabled": true
    }
  ],
  "token_saver": {
    "rtk": true,
    "caveman": false,
    "ponytail": "off"
  }
}
```

## Docker

```bash
docker run -d \
  -p 20128:20128 \
  -v ~/.7relay:/root/.7relay \
  --name 7relay \
  ghcr.io/medic1ne/7-relay:latest
```

## Roadmap

See [ROADMAP.md](ROADMAP.md) for full development plan.

## License

MIT
