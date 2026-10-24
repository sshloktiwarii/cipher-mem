# Foundry System Topology

Tags: #topology #architecture #platform

The Foundry Local AI PaaS interconnects 6 autonomous modules:
1. **[[foundry-router]]** (`:8080`): Dynamic Context Router & Token Proxy.
2. **[[foundry-crucible]]** (`:8081`): Ephemeral AI Execution Sandbox.
3. **[[foundry-pulse]]** (`:8082`): Telemetry & Anomaly Ingestion Engine.
4. **[[foundry-graph]]** (`:8083`): Obsidian Vector Engine & Knowledge Graph UI.
5. **[[foundry-airgap]]** (`:8085`): Stateless MCP Version Control Server with HITL.
6. **[[foundry-omni]]**: LangGraph Autonomous State Machine & Terminal UI.

Refer to [[architecture-decisions]] and [[incident-resolutions]] for operational standards.
