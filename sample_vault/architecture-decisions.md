# Architecture Decision Records (ADRs)

Tags: #architecture #decisions #foundry

## ADR-001: Context-Aware Dynamic Routing in [[foundry-router]]
- **Status**: Accepted
- **Context**: Large context prompts incur high latency and external API cost. Low token requests (< 1000 tokens) run faster and cost nothing when served by local Ollama instances.
- **Decision**: Route all requests with prompt tokens under 1000 to local Ollama on port 11434. Fallback to Cloud upstream if Ollama fails or token count exceeds budget.
- **Related Documents**: See [[incident-resolutions]] and [[system-topology]].

## ADR-002: Ephemeral Sandbox Isolation in [[foundry-crucible]]
- **Status**: Accepted
- **Context**: AI-generated code execution poses security and host stability risks.
- **Decision**: All untrusted scripts execute inside ephemeral Docker containers with `network_mode=none`, read-only root filesystems, 128MB RAM limits, 1 CPU core cap, and a strict 5-second SIGKILL timeout.
- **Related Documents**: See [[crucible-debugging]] and [[agent-playbooks]].
