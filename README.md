# cipher-mem: Obsidian Vector Engine & Memory Visualizer

A persistent memory engine and interactive force-directed graph visualizer built in Go and Next.js. It bridges local LLM agents directly with an Obsidian second brain, automatically parsing, chunking, and embedding Markdown vaults so agents can ground decisions in architecture records, incident resolutions, and team knowledge.

## Core Capabilities
- **Continuous Vault Watcher**: Uses `fsnotify` to track changes in any Obsidian vault in real time, auto-reindexing modified chunks and purging deleted notes.
- **Wikilink & Markdown Graph Extraction**: Resolves `[[wikilinks]]` and relative markdown links into a directed knowledge graph.
- **Hybrid Vector Embeddings**: Connects to local Ollama (`localhost:11434` with `nomic-embed-text`) with deterministic high-speed fallback vectors for offline development.
- **Sub-Millisecond Cosine Similarity**: Fast in-memory vector store with persistence to JSON/WAL snapshots.
- **Force-Directed Web UI**: Canvas-based 60fps graph simulation with physics springs, semantic search query highlighting, and document inspection drawer.

## API Endpoints
- `GET /health`: Engine status, indexed note count, total chunk count.
- `GET /api/graph`: D3-compatible nodes and links for force graph visualization.
- `POST /api/search`: Vector cosine search against vault memories (`{"query": "error in crucible", "top_k": 5}`).
- `POST /api/reindex`: Force immediate full vault re-scan.

## Quickstart

### 1. Run Backend Engine
```bash
go mod tidy
go run . --vault ./sample_vault --port 8083
```

### 2. Run Web Visualizer
```bash
cd web
npm install
npm run dev
# Open http://localhost:3003
```
