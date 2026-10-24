# 🧠 cipher-mem — Obsidian Vector Engine & Memory Visualizer

[![Language](https://img.shields.io/badge/Language-Go%201.24-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Status](https://img.shields.io/badge/Status-Production%20Ready-brightgreen?style=flat-square)](#)
[![Search Latency](https://img.shields.io/badge/Search%20Latency-%3C1ms-success?style=flat-square)](#)

A persistent second-brain memory engine and interactive force-directed graph visualizer built in Go. Bridges local AI agents with an Obsidian vault—automatically watching, parsing, chunking, and embedding Markdown notes so agents make decisions grounded in your actual team knowledge and incident resolutions.

---

### 💡 For Non-Techies: What does this do?
> AI models have short-term memory—they forget what happened yesterday or in past conversations.
>
> `cipher-mem` gives your AI a permanent memory library:
> - **Watches your notes folder**: Whenever you write or edit a note in Obsidian, `cipher-mem` reads and indexes it automatically.
> - **Understands meaning, not just keywords**: When an incident occurs, it retrieves relevant past solutions based on conceptual similarity (e.g., finding how you fixed a database timeout 3 months ago).
> - **Fast and private**: Searches your private notes in under 1 millisecond on your machine without uploading your files to any cloud.

---

## Core Capabilities

- **Continuous Vault Watcher**: Uses `fsnotify` to track changes in any Obsidian vault in real time, auto-reindexing modified chunks and purging deleted notes.
- **Wikilink & Markdown Graph Extraction**: Resolves `[[wikilinks]]` into a directed knowledge graph.
- **Sub-Millisecond Cosine Similarity**: High-speed in-memory vector store with persistence to JSON/WAL snapshots.
- **Zero-Port Sprawl**: Communicates over Unix Domain Socket (`/tmp/cipher/mem.sock`) or HTTP `:8083`.

---

## Quickstart

### Build Standalone Binary
```bash
go build -o cipher-mem .
```

### Run Memory Engine
```bash
./cipher-mem --vault ./sample_vault --port 8083 --socket /tmp/cipher/mem.sock
```

### Search Memories via Vector Similarity
```bash
curl -X POST http://localhost:8083/api/search \
  -H "Content-Type: application/json" \
  -d '{"query": "Execution Timeout Runaway Loop", "top_k": 3}'
```

---

## License
MIT © [Shlok Tiwari](https://github.com/sshloktiwarii)
