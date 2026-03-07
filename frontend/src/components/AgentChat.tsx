import { useState, useRef, useEffect, useCallback } from "react";
import Markdown from "react-markdown";
import { createAgentSession, sendAgentMessage, agentEventsURL } from "../api/client";

// A segment is one block in the agent's response stream, in arrival order.
type Segment =
  | { type: "text"; content: string }
  | { type: "reasoning"; content: string }
  | { type: "item"; item: AgentItem };

interface AgentItem {
  id: string;
  kind: string;
  status?: string;
  command?: string;
  output?: string;
  files?: string[];
  text?: string;
  exitCode?: number | null;
}

interface ChatMessage {
  id: string;
  role: "user" | "agent";
  segments: Segment[];
}

interface AgentChatProps {
  dashboardName: string;
  isOpen: boolean;
  onClose: () => void;
}

const THINKING_WORDS = [
  "Thinking", "Pondering", "Mulling", "Reasoning", "Considering",
  "Working", "Analyzing", "Processing", "Examining", "Figuring out",
];

export function AgentChat({ dashboardName, isOpen, onClose }: AgentChatProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [isStreaming, setIsStreaming] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [, setTurnId] = useState<string | null>(null);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const messageIdRef = useRef(0);

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages, scrollToBottom]);

  useEffect(() => {
    if (isOpen) {
      setTimeout(() => inputRef.current?.focus(), 100);
    }
  }, [isOpen]);

  useEffect(() => {
    return () => {
      eventSourceRef.current?.close();
    };
  }, []);

  const connectSSE = useCallback(
    (sid: string) => {
      eventSourceRef.current?.close();
      const es = new EventSource(agentEventsURL(sid));
      eventSourceRef.current = es;

      es.onmessage = (e) => {
        try {
          const data = JSON.parse(e.data);
          handleSSEEvent(data);
        } catch {
          // Ignore parse errors.
        }
      };

      es.onerror = () => {};
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );

  const updateAgent = useCallback(
    (updater: (segments: Segment[]) => Segment[]) => {
      setMessages((prev) => {
        const last = prev[prev.length - 1];
        if (last?.role === "agent") {
          return [
            ...prev.slice(0, -1),
            { ...last, segments: updater(last.segments) },
          ];
        }
        return [
          ...prev,
          {
            id: `msg-${++messageIdRef.current}`,
            role: "agent" as const,
            segments: updater([]),
          },
        ];
      });
    },
    [],
  );

  const handleSSEEvent = useCallback(
    (data: Record<string, unknown>) => {
      switch (data.type) {
        case "agent_delta": {
          const text = data.text as string;
          updateAgent((segs) => {
            const last = segs[segs.length - 1];
            if (last?.type === "text") {
              return [
                ...segs.slice(0, -1),
                { type: "text", content: last.content + text },
              ];
            }
            return [...segs, { type: "text", content: text }];
          });
          break;
        }

        case "reasoning_delta": {
          const text = data.text as string;
          updateAgent((segs) => {
            const last = segs[segs.length - 1];
            if (last?.type === "reasoning") {
              return [
                ...segs.slice(0, -1),
                { type: "reasoning", content: last.content + text },
              ];
            }
            return [...segs, { type: "reasoning", content: text }];
          });
          break;
        }

        case "item_started":
        case "item_completed": {
          const item = data.item as AgentItem;
          if (!item || item.kind === "agentMessage") break;

          // Reasoning items create/update a reasoning segment.
          if (item.kind === "reasoning") {
            if (data.type === "item_started") {
              updateAgent((segs) => {
                const last = segs[segs.length - 1];
                if (last?.type === "reasoning") return segs;
                return [...segs, { type: "reasoning" as const, content: "" }];
              });
            } else if (item.text) {
              // Backfill summary only if deltas didn't already populate it.
              updateAgent((segs) => {
                for (let i = segs.length - 1; i >= 0; i--) {
                  const s = segs[i];
                  if (s.type === "reasoning") {
                    if (!s.content) {
                      return [...segs.slice(0, i), { type: "reasoning" as const, content: item.text! }, ...segs.slice(i + 1)];
                    }
                    return segs;
                  }
                }
                return [...segs, { type: "reasoning" as const, content: item.text! }];
              });
            }
            break;
          }

          updateAgent((segs) => {
            const idx = segs.findIndex(
              (s) => s.type === "item" && s.item.id === item.id,
            );
            if (idx >= 0) {
              return segs.map((s, i) =>
                i === idx ? { type: "item" as const, item } : s,
              );
            }
            return [...segs, { type: "item", item }];
          });
          break;
        }

        case "command_output_delta": {
          const output = data.output as string;
          updateAgent((segs) => {
            for (let i = segs.length - 1; i >= 0; i--) {
              const s = segs[i];
              if (s.type === "item" && s.item.kind === "commandExecution") {
                const updated = {
                  ...s,
                  item: {
                    ...s.item,
                    output: (s.item.output ?? "") + output,
                  },
                };
                return [...segs.slice(0, i), updated, ...segs.slice(i + 1)];
              }
            }
            return segs;
          });
          break;
        }

        case "turn_started": {
          setTurnId(data.turn_id as string);
          break;
        }

        case "turn_completed": {
          setIsStreaming(false);
          setTurnId(null);
          break;
        }
      }
    },
    [updateAgent],
  );

  const createSession = useCallback(async (): Promise<string> => {
    const { session_id } = await createAgentSession(dashboardName);
    setSessionId(session_id);
    connectSSE(session_id);
    return session_id;
  }, [dashboardName, connectSSE]);

  const ensureSession = useCallback(async (): Promise<string> => {
    if (sessionId) return sessionId;
    return createSession();
  }, [sessionId, createSession]);

  const handleSend = useCallback(async () => {
    const text = input.trim();
    if (!text || isStreaming) return;

    setInput("");
    setError(null);

    const userMsg: ChatMessage = {
      id: `msg-${++messageIdRef.current}`,
      role: "user",
      segments: [{ type: "text", content: text }],
    };
    setMessages((prev) => [...prev, userMsg]);
    setIsStreaming(true);

    try {
      const sid = await ensureSession();
      try {
        await sendAgentMessage(sid, text);
      } catch (err) {
        // Thread may be stale (codex restarted) — retry with a fresh session.
        const msg = err instanceof Error ? err.message : "";
        if (msg.includes("thread not found") || msg.includes("500")) {
          setSessionId(null);
          const newSid = await createSession();
          await sendAgentMessage(newSid, text);
        } else {
          throw err;
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to send message");
      setIsStreaming(false);
    }
  }, [input, isStreaming, ensureSession, createSession]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className={`agent-sidebar ${isOpen ? "" : "agent-sidebar-closed"}`}>
      {/* Close button */}
      <button
        onClick={onClose}
        className="absolute top-2.5 right-2.5 z-10 w-6 h-6 flex items-center justify-center rounded hover:bg-[var(--dac-surface-hover)] text-[var(--dac-text-muted)] hover:text-[var(--dac-text-secondary)] transition-colors"
      >
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
          <path d="M4 4L12 12M12 4L4 12" />
        </svg>
      </button>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-4 relative">
        {messages.length === 0 && (
          <div className="absolute inset-0 flex flex-col">
            <WavyBackground />
            <div className="mt-auto px-4 pb-4">
              <p className="text-[12px] text-[var(--dac-text-muted)]">
                Describe changes to <span className="font-mono text-[var(--dac-text-secondary)]">{dashboardName}</span>
              </p>
            </div>
          </div>
        )}

        {messages.map((msg, idx) => (
          <div key={msg.id}>
            {msg.role === "user" ? (
              <UserMessage text={(msg.segments[0] as { content: string }).content} />
            ) : (
              <AgentMessage
                segments={msg.segments}
                isLastAndStreaming={isStreaming && idx === messages.length - 1}
              />
            )}
          </div>
        ))}

        {isStreaming && (() => {
          const last = messages[messages.length - 1];
          return !last || last.role !== "agent" || last.segments.length === 0;
        })() && (
          <div className="text-[12px] text-[var(--dac-text-muted)]">
            ...
          </div>
        )}

        {error && (
          <div className="text-[12px] text-[var(--dac-error)]">
            {error}
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div className="shrink-0 border-t border-[var(--dac-border)] px-3 py-2.5">
        <div className="relative">
          <textarea
            ref={inputRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Describe what to change..."
            rows={1}
            className="w-full resize-none rounded border border-[var(--dac-border)] bg-[var(--dac-surface)] text-[var(--dac-text-primary)] placeholder:text-[var(--dac-text-muted)] text-[13px] pl-3 pr-8 py-2 focus:outline-none focus:border-[var(--dac-text-muted)] transition-colors"
          />
          <button
            onClick={handleSend}
            disabled={!input.trim() || isStreaming}
            className="absolute right-2 top-1/2 -translate-y-1/2 w-6 h-6 flex items-center justify-center rounded text-[var(--dac-text-muted)] hover:text-[var(--dac-text-primary)] disabled:opacity-30 disabled:hover:text-[var(--dac-text-muted)] transition-colors"
          >
            <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
              <path d="M1.5 1.5L14.5 8L1.5 14.5V9.5L10 8L1.5 6.5V1.5Z" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}

function WavyBackground() {
  return (
    <div className="absolute inset-0 overflow-hidden pointer-events-none">
      <svg width="100%" height="100%" className="absolute inset-0">
        <defs>
          <filter id="grain">
            <feTurbulence type="fractalNoise" baseFrequency="0.65" numOctaves="3" stitchTiles="stitch" />
            <feColorMatrix type="saturate" values="0" />
          </filter>
          <linearGradient id="grain-fade" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="white" stopOpacity="0.1" />
            <stop offset="0.5" stopColor="white" stopOpacity="0.04" />
            <stop offset="1" stopColor="white" stopOpacity="0" />
          </linearGradient>
          <mask id="grain-mask">
            <rect width="100%" height="100%" fill="url(#grain-fade)" />
          </mask>
        </defs>
        <rect width="100%" height="100%" filter="url(#grain)" mask="url(#grain-mask)" opacity="1" />
      </svg>
      {/* Subtle accent gradient at the bottom */}
      <div
        className="absolute inset-x-0 top-0 h-[60%]"
        style={{
          background: `linear-gradient(to bottom, color-mix(in srgb, var(--dac-accent) 6%, transparent), transparent)`,
        }}
      />
    </div>
  );
}

function UserMessage({ text }: { text: string }) {
  return (
    <div className="flex justify-end">
      <div className="text-[13px] text-[var(--dac-text-primary)] leading-relaxed whitespace-pre-wrap bg-[var(--dac-surface-hover)] rounded-lg rounded-br-sm px-3 py-2 max-w-[85%]">
        {text}
      </div>
    </div>
  );
}

function AgentMessage({ segments, isLastAndStreaming }: { segments: Segment[]; isLastAndStreaming: boolean }) {
  // Find the last non-text segment to split process vs result.
  let lastNonTextIdx = -1;
  for (let i = segments.length - 1; i >= 0; i--) {
    if (segments[i].type !== "text") { lastNonTextIdx = i; break; }
  }

  let processSegs: Segment[];
  let resultSegs: Segment[];

  if (lastNonTextIdx === -1) {
    processSegs = [];
    resultSegs = segments;
  } else if (isLastAndStreaming) {
    processSegs = segments;
    resultSegs = [];
  } else {
    processSegs = segments.slice(0, lastNonTextIdx + 1);
    resultSegs = segments.slice(lastNonTextIdx + 1);
  }

  return (
    <div className="space-y-2">
      {processSegs.length > 0 && (
        <ThinkingBlock
          segments={processSegs}
          isActive={isLastAndStreaming && resultSegs.length === 0}
        />
      )}
      {resultSegs.map((seg, i) =>
        seg.type === "text" ? (
          <div key={`r-${i}`} className="agent-md text-[13px] text-[var(--dac-text-secondary)] leading-relaxed">
            <Markdown>{seg.content}</Markdown>
          </div>
        ) : null,
      )}
    </div>
  );
}

function formatDuration(secs: number): string {
  if (secs < 60) return `${secs} second${secs !== 1 ? "s" : ""}`;
  const mins = Math.round(secs / 60);
  return `${mins} minute${mins !== 1 ? "s" : ""}`;
}

function ThinkingBlock({ segments, isActive }: { segments: Segment[]; isActive: boolean }) {
  const [open, setOpen] = useState(true);
  const [wordIdx, setWordIdx] = useState(() => Math.floor(Math.random() * THINKING_WORDS.length));
  const [duration, setDuration] = useState<number | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const prevActive = useRef(isActive);
  const startTime = useRef(Date.now());

  // Auto-collapse when thinking finishes, record duration.
  useEffect(() => {
    if (prevActive.current && !isActive) {
      setOpen(false);
      setDuration(Math.round((Date.now() - startTime.current) / 1000));
    }
    prevActive.current = isActive;
  }, [isActive]);

  // Rotate the thinking word while active.
  useEffect(() => {
    if (!isActive) return;
    const interval = setInterval(() => {
      setWordIdx((prev) => {
        let next;
        do { next = Math.floor(Math.random() * THINKING_WORDS.length); } while (next === prev && THINKING_WORDS.length > 1);
        return next;
      });
    }, 3000);
    return () => clearInterval(interval);
  }, [isActive]);

  // Auto-scroll the substeps container.
  useEffect(() => {
    if (open && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [segments, open]);

  const title = duration !== null
    ? `Thought for ${formatDuration(duration)}`
    : THINKING_WORDS[wordIdx];

  return (
    <div className="text-[11px]">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1.5 text-[var(--dac-text-muted)] hover:text-[var(--dac-text-secondary)] transition-colors"
      >
        <svg
          width="8" height="8" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2"
          className={`transition-transform ${open ? "rotate-90" : ""}`}
        >
          <path d="M6 4L10 8L6 12" />
        </svg>
        <span className={isActive ? "agent-shimmer-text" : ""}>
          {title}
        </span>
      </button>
      {open && (
        <div
          ref={scrollRef}
          className="mt-1 pl-3 border-l border-[var(--dac-border)] space-y-1.5 max-h-[200px] overflow-y-auto">
          {segments.map((seg, i) => {
            if (seg.type === "text") {
              return (
                <div key={i} className="agent-md agent-md-muted text-[var(--dac-text-muted)] leading-relaxed">
                  <Markdown>{seg.content}</Markdown>
                </div>
              );
            }
            if (seg.type === "reasoning") {
              if (!seg.content) return null;
              return (
                <div key={i} className="text-[var(--dac-text-muted)] leading-relaxed whitespace-pre-wrap italic opacity-70">
                  {seg.content}
                </div>
              );
            }
            if (seg.type === "item") {
              return <ItemDisplay key={seg.item.id} item={seg.item} />;
            }
            return null;
          })}
        </div>
      )}
    </div>
  );
}

function ItemDisplay({ item }: { item: AgentItem }) {
  const [expanded, setExpanded] = useState(false);

  if (item.kind === "commandExecution") {
    if (!item.command) return null;
    return (
      <div className="rounded bg-[var(--dac-surface)] px-2 py-1.5">
        <button
          onClick={() => setExpanded(!expanded)}
          className="flex items-center gap-1.5 w-full text-[11px] text-[var(--dac-text-muted)] hover:text-[var(--dac-text-secondary)] transition-colors"
        >
          <StatusIcon status={item.status} exitCode={item.exitCode} />
          <span className="font-mono truncate text-left flex-1">
            {item.command}
          </span>
          <svg
            width="8" height="8" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2"
            className={`shrink-0 transition-transform ${expanded ? "rotate-90" : ""}`}
          >
            <path d="M6 4L10 8L6 12" />
          </svg>
        </button>
        {expanded && item.output && (
          <pre className="mt-1.5 pt-1.5 border-t border-[var(--dac-border)] text-[10px] font-mono text-[var(--dac-text-muted)] overflow-x-auto whitespace-pre-wrap">
            {item.output}
          </pre>
        )}
      </div>
    );
  }

  if (item.kind === "fileChange") {
    if (!item.files?.length) return null;
    return (
      <div className="flex items-center gap-1.5 text-[11px] text-[var(--dac-text-muted)] rounded bg-[var(--dac-surface)] px-2 py-1.5">
        <StatusIcon status={item.status} />
        <span className="font-mono truncate">
          {item.files?.map((f) => f.split("/").pop()).join(", ")}
        </span>
      </div>
    );
  }

  return null;
}

function StatusIcon({ status, exitCode }: { status?: string; exitCode?: number | null }) {
  if (status === "completed" || status === "applied") {
    const success = exitCode === undefined || exitCode === null || exitCode === 0;
    return (
      <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${success ? "bg-[var(--dac-success)]" : "bg-[var(--dac-error)]"}`} />
    );
  }
  if (status === "inProgress" || status === "running") {
    return <span className="w-1.5 h-1.5 rounded-full shrink-0 bg-[var(--dac-text-muted)] animate-pulse" />;
  }
  return <span className="w-1.5 h-1.5 rounded-full shrink-0 bg-[var(--dac-border)]" />;
}
