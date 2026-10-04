import type { UIMessage } from 'ai';

export type AgentKind = 'claude' | 'codex';

export type CoderEvent = {
  id: string;
  kind: 'say' | 'run' | 'read' | 'edit' | 'tool';
  text: string;
  exit?: number;
  output?: string;
};

/** Mirrors coderState in server/coder.go (streamed as a `data-coder` part). */
export type CoderState = {
  agent: string;
  status: 'starting' | 'running' | 'done' | 'error';
  events: CoderEvent[];
  filesChanged?: string[];
  diff?: string;
  summary?: string;
  error?: string;
  elapsedMs: number;
};

export type TestRun = { passed: boolean; output: string; durationMs: number };

export type DemoMessage = UIMessage<never, { coder: CoderState }>;

export type AgentStatus = {
  available: boolean;
  /** The environment variable that enables this pairing. */
  apiKeyEnv: string;
  chatModel: string;
  coder: string;
};

export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
