'use client';

import { useChat } from '@ai-sdk/react';
import { DefaultChatTransport, isToolUIPart, lastAssistantMessageIsCompleteWithApprovalResponses } from 'ai';
import Image from 'next/image';
import { useEffect, useMemo, useRef, useState } from 'react';
import { InlineText } from '@/components/InlineText';
import { Pipeline, type Phase } from '@/components/Pipeline';
import { ToolPart } from '@/components/ToolPart';
import { API_URL, type AgentKind, type AgentStatus, type CoderState, type DemoMessage } from '@/lib/types';

const SUGGESTION = 'CI is red on shortlink. Find out why and get it fixed.';

const PAIRINGS: { kind: AgentKind; label: string }[] = [
  { kind: 'claude', label: 'Claude Code' },
  { kind: 'codex', label: 'Codex' },
];

export default function Page() {
  const [agent, setAgent] = useState<AgentKind>('claude');
  const [status, setStatus] = useState<Record<AgentKind, AgentStatus> | null>(null);
  const [input, setInput] = useState('');
  const agentRef = useRef(agent);
  agentRef.current = agent;

  useEffect(() => {
    fetch(`${API_URL}/api/status`)
      .then((r) => r.json())
      .then((s: { agents: Record<AgentKind, AgentStatus> }) => {
        setStatus(s.agents);
        // Start on the first pairing that has an API key configured.
        const first = PAIRINGS.find((p) => s.agents[p.kind]?.available);
        if (first && !s.agents[agentRef.current]?.available) setAgent(first.kind);
      })
      .catch((err) => console.error('GET /api/status failed', err));
  }, []);

  // The transport reads the current pairing on every request, including the
  // automatic resend after an approval.
  const transport = useMemo(
    () => new DefaultChatTransport<DemoMessage>({ api: `${API_URL}/api/chat`, body: () => ({ agent: agentRef.current }) }),
    [],
  );

  const { messages, sendMessage, status: chatStatus, addToolApprovalResponse, setMessages, error, stop } =
    useChat<DemoMessage>({
      transport,
      sendAutomaticallyWhen: lastAssistantMessageIsCompleteWithApprovalResponses,
    });

  const busy = chatStatus === 'submitted' || chatStatus === 'streaming';
  const phase = derivePhase(messages, chatStatus);
  const current = status?.[agent];

  const threadEnd = useRef<HTMLDivElement>(null);
  useEffect(() => {
    threadEnd.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
  }, [messages]);

  function send(text: string) {
    if (!text.trim() || busy) return;
    sendMessage({ text });
    setInput('');
  }

  async function startOver() {
    stop();
    await fetch(`${API_URL}/api/reset`, { method: 'POST' }).catch(() => {});
    setMessages([]);
  }

  return (
    <div className="flex h-dvh flex-col">
      <header className="border-b-2 border-ink bg-white">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-3 sm:px-6">
          <div className="flex items-center gap-3">
            <Image src="/logo.png" alt="go-ai" width={57} height={44} priority />
            <div>
              <h1 className="text-[19px] font-extrabold leading-none tracking-tight">Shipyard</h1>
              <p className="mt-0.5 hidden text-[12.5px] text-muted sm:block">Fixes failing Go tests. The reference app for the Go AI SDK.</p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <div role="radiogroup" aria-label="Coding agent" className="flex rounded-xl border-2 border-ink bg-paper p-0.5">
              {PAIRINGS.map((p) => {
                const available = status?.[p.kind]?.available ?? true;
                return (
                  <button
                    key={p.kind}
                    role="radio"
                    aria-checked={agent === p.kind}
                    disabled={!available || busy}
                    title={available ? undefined : `Add ${status?.[p.kind]?.apiKeyEnv ?? 'its API key'} to .env and restart the server`}
                    onClick={() => setAgent(p.kind)}
                    className={[
                      'rounded-[9px] px-3 py-1.5 text-[14px] font-bold whitespace-nowrap transition-colors disabled:cursor-not-allowed disabled:opacity-40',
                      agent === p.kind ? 'bg-ink text-white' : 'hover:bg-gopher-soft',
                    ].join(' ')}
                  >
                    {p.label}
                  </button>
                );
              })}
            </div>
            <button
              type="button"
              onClick={startOver}
              className="rounded-xl border-2 border-ink bg-white px-3 py-1.5 text-[14px] font-bold whitespace-nowrap hover:bg-gopher-soft"
            >
              Start over
            </button>
          </div>
        </div>
      </header>

      <div className="hidden justify-center border-b-2 border-ink bg-gopher-soft px-4 py-3 md:flex">
        <Pipeline phase={phase} chatModel={current?.chatModel ?? 'chat model'} coder={current?.coder ?? 'Coding agent'} />
      </div>

      <main className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-[760px] space-y-5 px-4 py-8 sm:px-6">
          {messages.length === 0 && <EmptyState onPick={send} coder={current?.coder ?? 'a coding agent'} />}

          {messages.map((message) => (
            <MessageView
              key={message.id}
              message={message}
              coderName={current?.coder ?? 'the coding agent'}
              onApproval={(id, approved) => addToolApprovalResponse({ id, approved })}
            />
          ))}

          {error && (
            <p className="rounded-xl border-2 border-ink bg-fail/10 px-4 py-3 text-[14px] text-[#a3242a]">
              {isNetworkError(error)
                ? `Can't reach the server at ${API_URL}. Start it with make dev.`
                : `Something went wrong: ${error.message}`}
            </p>
          )}
          <div ref={threadEnd} />
        </div>
      </main>

      <footer className="border-t-2 border-ink bg-white">
        <form
          className="mx-auto flex max-w-[760px] gap-3 px-4 py-3 sm:px-6"
          onSubmit={(e) => {
            e.preventDefault();
            send(input);
          }}
        >
          <label htmlFor="prompt" className="sr-only">
            Message
          </label>
          <input
            id="prompt"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Describe what's broken"
            autoComplete="off"
            className="min-w-0 flex-1 rounded-xl border-2 border-ink bg-paper px-4 py-2.5 text-[15px] placeholder:text-muted focus:bg-white focus:outline-none"
          />
          {busy ? (
            <button type="button" onClick={stop} className="rounded-xl border-2 border-ink bg-white px-5 font-bold">
              Stop
            </button>
          ) : (
            <button
              type="submit"
              disabled={!input.trim()}
              className="rounded-xl border-2 border-ink bg-go px-5 font-bold text-white shadow-ink hover:bg-go-deep disabled:opacity-40 disabled:shadow-none disabled:hover:bg-go"
            >
              Send
            </button>
          )}
        </form>
        <p className="px-4 pb-2 text-center text-[11px] text-muted">
          Go is a trademark of Google. The Go gopher, whenever used, is an original creation by Renée French.
        </p>
      </footer>
    </div>
  );
}

// fetch rejects with a TypeError when the server can't be reached; the
// message differs by browser ("Failed to fetch", "Load failed", ...).
function isNetworkError(error: Error) {
  return error instanceof TypeError || /failed to fetch|load failed|networkerror/i.test(error.message);
}

function EmptyState({ onPick, coder }: { onPick: (text: string) => void; coder: string }) {
  return (
    <div className="pt-10">
      <h2 className="max-w-[22ch] text-[34px] font-extrabold leading-[1.1] tracking-tight">
        Fix the failing shortlink tests.
      </h2>
      <p className="mt-3 max-w-[60ch] text-[16px] leading-relaxed text-muted">
        The <code className="font-mono text-ink">shortlink</code> package has two failing tests. Shipyard runs them, finds
        the cause, and asks for your approval before {coder} edits any code.
      </p>
      <button
        type="button"
        onClick={() => onPick(SUGGESTION)}
        className="mt-6 rounded-xl border-2 border-ink bg-white px-4 py-3 text-left text-[16px] font-semibold shadow-ink hover:bg-gopher-soft"
      >
        {SUGGESTION}
      </button>
    </div>
  );
}

function MessageView({
  message,
  coderName,
  onApproval,
}: {
  message: DemoMessage;
  coderName: string;
  onApproval: (id: string, approved: boolean) => void;
}) {
  if (message.role === 'user') {
    const text = message.parts.map((p) => (p.type === 'text' ? p.text : '')).join('');
    return (
      <div className="flex justify-end">
        <p className="max-w-[80%] rounded-2xl rounded-br-md border-2 border-ink bg-gopher px-4 py-2.5 text-[16px] font-medium">
          {text}
        </p>
      </div>
    );
  }

  // Coding-agent progress arrives as `data-coder` parts keyed by tool call ID.
  const coderById = new Map<string, CoderState>();
  for (const part of message.parts) {
    if (part.type === 'data-coder' && part.id) coderById.set(part.id, part.data);
  }

  return (
    <div className="space-y-3">
      {message.parts.map((part, i) => {
        if (part.type === 'text') {
          return part.text ? (
            <p key={i} className="whitespace-pre-wrap text-[16px] leading-relaxed">
              <InlineText
                text={part.text}
                codeClassName="rounded-md border border-ink/15 bg-white px-1 py-px font-mono text-[14.5px]"
              />
            </p>
          ) : null;
        }
        if (isToolUIPart(part)) {
          return (
            <ToolPart
              key={part.toolCallId}
              part={part}
              coder={coderById.get(part.toolCallId)}
              coderName={coderName}
              onApproval={onApproval}
            />
          );
        }
        return null;
      })}
    </div>
  );
}

function derivePhase(messages: DemoMessage[], chatStatus: string): Phase {
  const last = messages.at(-1);
  if (last?.role === 'assistant') {
    const awaiting = last.parts.some((p) => isToolUIPart(p) && p.state === 'approval-requested');
    if (awaiting && chatStatus === 'ready') return 'approval';
    const coding = last.parts.some(
      (p) => p.type === 'data-coder' && (p.data.status === 'starting' || p.data.status === 'running'),
    );
    if (coding && chatStatus === 'streaming') return 'coding';
  }
  if (chatStatus === 'submitted') return 'sending';
  if (chatStatus === 'streaming') return 'thinking';
  return 'idle';
}
