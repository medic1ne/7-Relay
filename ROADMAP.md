# 7-Relay Roadmap

## Phase 1: Core Router (Week 1) 🔴 CURRENT

- [x] Project structure + Go module
- [x] Config system (JSON-based)
- [x] HTTP server + OpenAI-compatible endpoint
- [x] Provider pool + basic forwarding
- [x] OpenAI/Claude/Gemini provider adapters
- [ ] Request/response format translation
- [ ] Error handling + retry logic
- [ ] Basic unit tests

## Phase 2: Smart Routing (Week 1-2)

- [ ] 3-tier fallback engine
- [ ] Provider health checks (ping endpoint)
- [ ] Circuit breaker (auto-disable unhealthy)
- [ ] Rate limiter (token bucket)
- [ ] Multi-account round-robin
- [ ] Quota tracking per provider
- [ ] Combo system (model groups)

## Phase 3: Token Savers (Week 2)

- [ ] RTK compression (tool output detection)
- [ ] Git diff compression
- [ ] Grep output compression
- [ ] Caveman mode (terse output)
- [ ] Ponytail mode (YAGNI prompt injection)
- [ ] Headroom integration hook
- [ ] Token usage analytics

## Phase 4: Auth & Security (Week 2-3)

- [ ] OAuth flow (Claude/Codex/Cursor)
- [ ] API key management
- [ ] E2E encrypted key vault (ChaCha20-Poly1305)
- [ ] MITM proxy (HTTP/HTTPS/WebSocket)
- [ ] TLS certificate generation
- [ ] Request signing

## Phase 5: TUI Dashboard (Week 3)

- [ ] Bubble Tea TUI framework
- [ ] Provider management view
- [ ] Connection/config editor
- [ ] Real-time usage stats
- [ ] Combo builder
- [ ] Settings menu

## Phase 6: Local Inference (Week 3-4)

- [ ] Ollama integration
- [ ] llama.cpp integration
- [ ] Local model discovery
- [ ] Hybrid routing (cloud + local)
- [ ] Offline mode

## Phase 7: Extras (Week 4)

- [ ] Session persistence + context cache
- [ ] Plugin system (WASM)
- [ ] Docker multi-stage build
- [ ] Binary delta updates
- [ ] Web3 wallet auth (optional)
- [ ] Documentation site
- [ ] Video tutorials

---

## Completed Milestones

| Milestone | Date | Status |
|---|---|---|
| v0.1.0 - Core Router | - | 🟡 In Progress |
| v0.2.0 - Smart Routing | - | ⚪ Not Started |
| v0.3.0 - Token Savers | - | ⚪ Not Started |
| v0.4.0 - Auth & Security | - | ⚪ Not Started |
| v0.5.0 - TUI Dashboard | - | ⚪ Not Started |
| v1.0.0 - Full Release | - | ⚪ Not Started |
