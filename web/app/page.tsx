"use client";

import React, { useEffect, useRef, useState, useMemo } from "react";
import { Search, RefreshCw, Cpu, Database, Network, ChevronRight, FileText, ExternalLink, Zap } from "lucide-react";

interface GraphNode {
  id: string;
  name: string;
  path: string;
  group: string;
  tags: string[];
  size: number;
  chunk_count: number;
  x?: number;
  y?: number;
  vx?: number;
  vy?: number;
}

interface GraphLink {
  source: string;
  target: string;
  value: number;
  type: string;
}

interface GraphData {
  nodes: GraphNode[];
  links: GraphLink[];
}

interface SearchResult {
  chunk: {
    id: string;
    file: string;
    title: string;
    heading: string;
    content: string;
    tags: string[];
  };
  similarity: number;
}

export default function FoundryGraphPage() {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const [graphData, setGraphData] = useState<GraphData>({ nodes: [], links: [] });
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [isReindexing, setIsReindexing] = useState(false);
  const [stats, setStats] = useState({ files: 0, chunks: 0, status: "connecting" });

  const simulationRef = useRef<{
    nodes: GraphNode[];
    links: { sourceNode: GraphNode; targetNode: GraphNode }[];
    transform: { x: number; y: number; k: number };
    draggingNode: GraphNode | null;
  }>({
    nodes: [],
    links: [],
    transform: { x: 0, y: 0, k: 1 },
    draggingNode: null,
  });

  // Fetch graph data & stats
  const fetchGraph = async () => {
    try {
      const [graphRes, healthRes] = await Promise.all([
        fetch("http://localhost:8083/api/graph").catch(() => null),
        fetch("http://localhost:8083/health").catch(() => null),
      ]);

      if (graphRes && graphRes.ok) {
        const data: GraphData = await graphRes.json();
        setGraphData(data);
      }
      if (healthRes && healthRes.ok) {
        const hData = await healthRes.json();
        setStats({ files: hData.indexed_files, chunks: hData.total_chunks, status: "connected" });
      } else {
        setStats(s => ({ ...s, status: "offline" }));
      }
    } catch {
      setStats(s => ({ ...s, status: "offline" }));
    }
  };

  useEffect(() => {
    fetchGraph();
    const interval = setInterval(fetchGraph, 10000);
    return () => clearInterval(interval);
  }, []);

  // Initialize Force Simulation on canvas
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || graphData.nodes.length === 0) return;

    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const width = canvas.clientWidth;
    const height = canvas.clientHeight;
    canvas.width = width * window.devicePixelRatio;
    canvas.height = height * window.devicePixelRatio;
    ctx.scale(window.devicePixelRatio, window.devicePixelRatio);

    // Position nodes randomly in circle
    const nodes: GraphNode[] = graphData.nodes.map((n, i) => {
      const angle = (i / graphData.nodes.length) * 2 * Math.PI;
      const radius = 150 + Math.random() * 80;
      return {
        ...n,
        x: width / 2 + Math.cos(angle) * radius,
        y: height / 2 + Math.sin(angle) * radius,
        vx: 0,
        vy: 0,
      };
    });

    const nodeById = new Map<string, GraphNode>();
    nodes.forEach(n => nodeById.set(n.id, n));

    const links = graphData.links
      .map(l => ({
        sourceNode: nodeById.get(l.source) || nodeById.get(nodes.find(n => n.name === l.source)?.id || ""),
        targetNode: nodeById.get(l.target) || nodeById.get(nodes.find(n => n.name === l.target)?.id || ""),
      }))
      .filter((l): l is { sourceNode: GraphNode; targetNode: GraphNode } => !!(l.sourceNode && l.targetNode));

    simulationRef.current = {
      nodes,
      links,
      transform: { x: 0, y: 0, k: 1 },
      draggingNode: null,
    };

    let animationFrameId: number;

    const tick = () => {
      // 1. Physics update
      const repulse = 800;
      const springLength = 120;
      const springStrength = 0.04;
      const centerStrength = 0.02;

      for (let i = 0; i < nodes.length; i++) {
        const a = nodes[i];
        if (a === simulationRef.current.draggingNode) continue;

        // Repulsion
        for (let j = i + 1; j < nodes.length; j++) {
          const b = nodes[j];
          const dx = (b.x || 0) - (a.x || 0);
          const dy = (b.y || 0) - (a.y || 0);
          const distSq = dx * dx + dy * dy || 1;
          const dist = Math.sqrt(distSq);
          const force = repulse / distSq;
          const fx = (dx / dist) * force;
          const fy = (dy / dist) * force;
          a.vx = (a.vx || 0) - fx;
          a.vy = (a.vy || 0) - fy;
          b.vx = (b.vx || 0) + fx;
          b.vy = (b.vy || 0) + fy;
        }

        // Gravity to center
        const cdx = width / 2 - (a.x || 0);
        const cdy = height / 2 - (a.y || 0);
        a.vx = (a.vx || 0) + cdx * centerStrength;
        a.vy = (a.vy || 0) + cdy * centerStrength;
      }

      // Spring attraction
      links.forEach(l => {
        const dx = (l.targetNode.x || 0) - (l.sourceNode.x || 0);
        const dy = (l.targetNode.y || 0) - (l.sourceNode.y || 0);
        const dist = Math.sqrt(dx * dx + dy * dy) || 1;
        const delta = dist - springLength;
        const fx = (dx / dist) * delta * springStrength;
        const fy = (dy / dist) * delta * springStrength;

        if (l.sourceNode !== simulationRef.current.draggingNode) {
          l.sourceNode.vx = (l.sourceNode.vx || 0) + fx;
          l.sourceNode.vy = (l.sourceNode.vy || 0) + fy;
        }
        if (l.targetNode !== simulationRef.current.draggingNode) {
          l.targetNode.vx = (l.targetNode.vx || 0) - fx;
          l.targetNode.vy = (l.targetNode.vy || 0) - fy;
        }
      });

      // Position update with damping
      const damping = 0.85;
      nodes.forEach(n => {
        if (n !== simulationRef.current.draggingNode) {
          n.vx = (n.vx || 0) * damping;
          n.vy = (n.vy || 0) * damping;
          n.x = (n.x || 0) + n.vx;
          n.y = (n.y || 0) + n.vy;
        }
      });

      // 2. Render
      ctx.clearRect(0, 0, width, height);
      const { x: tx, y: ty, k } = simulationRef.current.transform;

      ctx.save();
      ctx.translate(tx, ty);
      ctx.scale(k, k);

      // Draw links
      ctx.lineWidth = 1;
      links.forEach(l => {
        ctx.strokeStyle = "rgba(255, 255, 255, 0.12)";
        ctx.beginPath();
        ctx.moveTo(l.sourceNode.x || 0, l.sourceNode.y || 0);
        ctx.lineTo(l.targetNode.x || 0, l.targetNode.y || 0);
        ctx.stroke();
      });

      // Draw nodes
      nodes.forEach(n => {
        const isSelected = selectedNode?.id === n.id;
        const isMatched = searchResults.some(r => r.chunk.file === n.path);
        const r = Math.max(6, n.size * 0.75);

        // Glow
        if (isMatched || isSelected) {
          ctx.beginPath();
          ctx.arc(n.x || 0, n.y || 0, r + 8, 0, 2 * Math.PI);
          ctx.fillStyle = isMatched ? "rgba(245, 158, 11, 0.3)" : "rgba(59, 130, 246, 0.3)";
          ctx.fill();
        }

        ctx.beginPath();
        ctx.arc(n.x || 0, n.y || 0, r, 0, 2 * Math.PI);
        ctx.fillStyle = isMatched ? "#f59e0b" : isSelected ? "#3b82f6" : "#10b981";
        ctx.fill();

        // Node Label
        ctx.fillStyle = "#e4e4e7";
        ctx.font = "11px 'JetBrains Mono', monospace";
        ctx.textAlign = "center";
        ctx.fillText(n.name, n.x || 0, (n.y || 0) + r + 14);
      });

      ctx.restore();
      animationFrameId = requestAnimationFrame(tick);
    };

    tick();

    // Mouse Interactions (Pan, Drag, Select)
    let isPanning = false;
    let startX = 0;
    let startY = 0;

    const onMouseDown = (e: MouseEvent) => {
      const rect = canvas.getBoundingClientRect();
      const mouseX = e.clientX - rect.left;
      const mouseY = e.clientY - rect.top;

      const { x: tx, y: ty, k } = simulationRef.current.transform;
      const worldX = (mouseX - tx) / k;
      const worldY = (mouseY - ty) / k;

      // Check node hit
      const hit = nodes.find(n => {
        const dx = (n.x || 0) - worldX;
        const dy = (n.y || 0) - worldY;
        return Math.sqrt(dx * dx + dy * dy) <= Math.max(10, n.size * 0.75);
      });

      if (hit) {
        simulationRef.current.draggingNode = hit;
        setSelectedNode(hit);
      } else {
        isPanning = true;
        startX = mouseX - tx;
        startY = mouseY - ty;
        setSelectedNode(null);
      }
    };

    const onMouseMove = (e: MouseEvent) => {
      const rect = canvas.getBoundingClientRect();
      const mouseX = e.clientX - rect.left;
      const mouseY = e.clientY - rect.top;

      if (simulationRef.current.draggingNode) {
        const { x: tx, y: ty, k } = simulationRef.current.transform;
        simulationRef.current.draggingNode.x = (mouseX - tx) / k;
        simulationRef.current.draggingNode.y = (mouseY - ty) / k;
      } else if (isPanning) {
        simulationRef.current.transform.x = mouseX - startX;
        simulationRef.current.transform.y = mouseY - startY;
      }
    };

    const onMouseUp = () => {
      simulationRef.current.draggingNode = null;
      isPanning = false;
    };

    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      const zoomFactor = e.deltaY < 0 ? 1.1 : 0.9;
      simulationRef.current.transform.k = Math.max(0.3, Math.min(3, simulationRef.current.transform.k * zoomFactor));
    };

    const onResize = () => {
      if (!canvas) return;
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      canvas.width = w * window.devicePixelRatio;
      canvas.height = h * window.devicePixelRatio;
      ctx.scale(window.devicePixelRatio, window.devicePixelRatio);
    };

    window.addEventListener("resize", onResize);
    canvas.addEventListener("mousedown", onMouseDown);
    window.addEventListener("mousemove", onMouseMove);
    window.addEventListener("mouseup", onMouseUp);
    canvas.addEventListener("wheel", onWheel);

    return () => {
      cancelAnimationFrame(animationFrameId);
      window.removeEventListener("resize", onResize);
      canvas.removeEventListener("mousedown", onMouseDown);
      window.removeEventListener("mousemove", onMouseMove);
      window.removeEventListener("mouseup", onMouseUp);
      canvas.removeEventListener("wheel", onWheel);
    };
  }, [graphData, selectedNode, searchResults]);

  // Handle Semantic Vector Search
  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!searchQuery.trim()) return;

    setIsSearching(true);
    try {
      const res = await fetch("http://localhost:8083/api/search", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: searchQuery, top_k: 5 }),
      });
      if (res.ok) {
        const data = await res.json();
        setSearchResults(data.results || []);
      }
    } catch (err) {
      console.error("Search failed:", err);
    } finally {
      setIsSearching(false);
    }
  };

  const handleReindex = async () => {
    setIsReindexing(true);
    try {
      await fetch("http://localhost:8083/api/reindex", { method: "POST" });
      setTimeout(fetchGraph, 1000);
    } finally {
      setIsReindexing(false);
    }
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100vh", overflow: "hidden" }}>
      {/* Top Navbar */}
      <header
        style={{
          height: 64,
          borderBottom: "1px solid #27272a",
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          padding: "0 24px",
          background: "#09090b",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <div style={{ padding: 8, background: "#18181b", borderRadius: 8, border: "1px solid #27272a" }}>
            <Network size={20} color="#10b981" />
          </div>
          <div>
            <h1 style={{ fontSize: 16, fontWeight: 700, margin: 0, letterSpacing: "-0.02em" }}>
              Foundry Graph
            </h1>
            <p style={{ fontSize: 12, color: "#a1a1aa", margin: 0 }}>
              Obsidian Vector Engine & Memory Visualizer
            </p>
          </div>
        </div>

        {/* Search Bar */}
        <form onSubmit={handleSearch} style={{ display: "flex", alignItems: "center", gap: 8, minWidth: 420 }}>
          <div style={{ position: "relative", width: "100%" }}>
            <Search size={16} color="#71717a" style={{ position: "absolute", left: 12, top: 12 }} />
            <input
              type="text"
              placeholder="Search vault memories (e.g. 'Crucible timeout', 'ADR routing')..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              style={{
                width: "100%",
                padding: "10px 12px 10px 38px",
                backgroundColor: "#18181b",
                border: "1px solid #27272a",
                borderRadius: 8,
                color: "#f4f4f5",
                fontSize: 13,
                outline: "none",
                fontFamily: "inherit",
              }}
            />
          </div>
          <button
            type="submit"
            disabled={isSearching}
            style={{
              padding: "10px 16px",
              background: "#10b981",
              color: "#000",
              fontWeight: 600,
              fontSize: 13,
              border: "none",
              borderRadius: 8,
              cursor: "pointer",
              display: "flex",
              alignItems: "center",
              gap: 6,
            }}
          >
            {isSearching ? <RefreshCw size={14} className="spin" /> : <Zap size={14} />}
            Search
          </button>
        </form>

        {/* Telemetry Stats */}
        <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 12, color: "#a1a1aa" }}>
            <FileText size={14} color="#10b981" />
            <span>{stats.files} Files</span>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 12, color: "#a1a1aa" }}>
            <Database size={14} color="#3b82f6" />
            <span>{stats.chunks} Chunks</span>
          </div>
          <button
            onClick={handleReindex}
            disabled={isReindexing}
            title="Re-index Vault"
            style={{
              padding: "8px 12px",
              background: "#18181b",
              border: "1px solid #27272a",
              color: "#e4e4e7",
              borderRadius: 6,
              cursor: "pointer",
              display: "flex",
              alignItems: "center",
              gap: 6,
              fontSize: 12,
            }}
          >
            <RefreshCw size={14} />
            Reindex
          </button>
        </div>
      </header>

      {/* Main Graph Area */}
      <div style={{ flex: 1, position: "relative", overflow: "hidden" }}>
        <canvas
          ref={canvasRef}
          style={{ width: "100%", height: "100%", cursor: "grab", display: "block" }}
        />

        {/* Node Detail Drawer */}
        {selectedNode && (
          <aside
            style={{
              position: "absolute",
              right: 24,
              top: 24,
              bottom: 24,
              width: 380,
              background: "rgba(24, 24, 27, 0.95)",
              backdropFilter: "blur(12px)",
              border: "1px solid #27272a",
              borderRadius: 12,
              padding: 20,
              display: "flex",
              flexDirection: "column",
              gap: 16,
              boxShadow: "0 20px 40px rgba(0,0,0,0.5)",
              overflowY: "auto",
            }}
          >
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
              <div>
                <span
                  style={{
                    fontSize: 10,
                    textTransform: "uppercase",
                    letterSpacing: "0.08em",
                    padding: "3px 8px",
                    background: "#27272a",
                    borderRadius: 4,
                    color: "#a1a1aa",
                  }}
                >
                  {selectedNode.group}
                </span>
                <h3 style={{ fontSize: 18, fontWeight: 700, margin: "8px 0 4px 0" }}>
                  {selectedNode.name}
                </h3>
                <p style={{ fontSize: 12, color: "#71717a", margin: 0, fontFamily: "monospace" }}>
                  {selectedNode.path}
                </p>
              </div>
              <button
                onClick={() => setSelectedNode(null)}
                style={{ background: "transparent", border: "none", color: "#a1a1aa", cursor: "pointer", fontSize: 16 }}
              >
                ✕
              </button>
            </div>

            <div style={{ borderTop: "1px solid #27272a", paddingTop: 12 }}>
              <h4 style={{ fontSize: 12, fontWeight: 600, color: "#a1a1aa", marginBottom: 8 }}>
                Memory Metadata
              </h4>
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8, fontSize: 12 }}>
                <div style={{ background: "#09090b", padding: 8, borderRadius: 6, border: "1px solid #27272a" }}>
                  <div style={{ color: "#71717a" }}>Semantic Chunks</div>
                  <div style={{ fontSize: 14, fontWeight: 600, color: "#f4f4f5" }}>{selectedNode.chunk_count}</div>
                </div>
                <div style={{ background: "#09090b", padding: 8, borderRadius: 6, border: "1px solid #27272a" }}>
                  <div style={{ color: "#71717a" }}>Tags</div>
                  <div style={{ fontSize: 12, fontWeight: 500, color: "#10b981" }}>
                    {selectedNode.tags.length > 0 ? selectedNode.tags.join(", ") : "none"}
                  </div>
                </div>
              </div>
            </div>

            {/* Semantic Matches if node appears in search */}
            {searchResults.filter(r => r.chunk.file === selectedNode.path).length > 0 && (
              <div style={{ borderTop: "1px solid #27272a", paddingTop: 12 }}>
                <h4 style={{ fontSize: 12, fontWeight: 600, color: "#f59e0b", marginBottom: 8 }}>
                  Search Matches in Document
                </h4>
                {searchResults
                  .filter(r => r.chunk.file === selectedNode.path)
                  .map(match => (
                    <div
                      key={match.chunk.id}
                      style={{
                        background: "#09090b",
                        border: "1px solid #3f3f46",
                        borderRadius: 6,
                        padding: 10,
                        marginBottom: 8,
                      }}
                    >
                      <div style={{ display: "flex", justifyContent: "space-between", fontSize: 11, marginBottom: 4 }}>
                        <span style={{ fontWeight: 600, color: "#e4e4e7" }}>{match.chunk.heading}</span>
                        <span style={{ color: "#f59e0b" }}>{(match.similarity * 100).toFixed(1)}% match</span>
                      </div>
                      <p style={{ fontSize: 11, color: "#a1a1aa", margin: 0, lineHeight: 1.4 }}>
                        {match.chunk.content}
                      </p>
                    </div>
                  ))}
              </div>
            )}
          </aside>
        )}

        {/* Search Results Drawer */}
        {searchResults.length > 0 && !selectedNode && (
          <aside
            style={{
              position: "absolute",
              left: 24,
              bottom: 24,
              maxHeight: 280,
              width: 440,
              background: "rgba(24, 24, 27, 0.95)",
              backdropFilter: "blur(12px)",
              border: "1px solid #27272a",
              borderRadius: 12,
              padding: 16,
              overflowY: "auto",
              boxShadow: "0 20px 40px rgba(0,0,0,0.5)",
            }}
          >
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12 }}>
              <span style={{ fontSize: 12, fontWeight: 700, color: "#f59e0b" }}>
                Top Vector Matches ({searchResults.length})
              </span>
              <button
                onClick={() => setSearchResults([])}
                style={{ background: "transparent", border: "none", color: "#a1a1aa", cursor: "pointer", fontSize: 12 }}
              >
                Dismiss
              </button>
            </div>
            {searchResults.map((res, i) => (
              <div
                key={res.chunk.id}
                onClick={() => {
                  const node = simulationRef.current.nodes.find(n => n.path === res.chunk.file);
                  if (node) setSelectedNode(node);
                }}
                style={{
                  background: "#09090b",
                  border: "1px solid #27272a",
                  borderRadius: 6,
                  padding: 10,
                  marginBottom: 8,
                  cursor: "pointer",
                }}
              >
                <div style={{ display: "flex", justifyContent: "space-between", fontSize: 12, fontWeight: 600 }}>
                  <span style={{ color: "#e4e4e7" }}>{res.chunk.title}</span>
                  <span style={{ color: "#f59e0b" }}>{(res.similarity * 100).toFixed(1)}%</span>
                </div>
                <div style={{ fontSize: 11, color: "#71717a", marginBottom: 4 }}>{res.chunk.heading}</div>
                <p style={{ fontSize: 11, color: "#a1a1aa", margin: 0, lineClamp: 2, overflow: "hidden" }}>
                  {res.chunk.content}
                </p>
              </div>
            ))}
          </aside>
        )}
      </div>
    </div>
  );
}
