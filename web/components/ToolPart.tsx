'use client';

import type { DynamicToolUIPart, ToolUIPart } from 'ai';
import { getToolName } from 'ai';
import type { CoderState, TestRun } from '@/lib/types';
import { CoderCard } from './CoderCard';

type AnyToolPart = ToolUIPart | DynamicToolUIPart;

function Row({ children, tone = 'plain' }: { children: React.ReactNode; tone?: 'plain' | 'pass' | 'fail' | 'busy' }) {
  const dot = { plain: 'bg-white', pass: 'bg-pass', fail: 'bg-fail', busy: 'animate-pulse bg-go' }[tone];
  return (
    <div className="flex items-center gap-2.5 text-[14px] text-muted">
      <span aria-hidden className={`size-2.5 shrink-0 rounded-full border-2 border-ink ${dot}`} />
      {children}
    </div>
  );
}

function TestsPart({ part }: { part: AnyToolPart }) {
  if (part.state !== 'output-available') {
    return (
      <Row tone="busy">
        Running <code className="font-mono text-ink">go test ./...</code>
      </Row>
    );
  }
  const run = part.output as TestRun;
  const failures = run.output.match(/^--- FAIL/gm)?.length ?? 0;
  return (
    <details className="group">
      <summary className="cursor-pointer list-none [&::-webkit-details-marker]:hidden">
        <Row tone={run.passed ? 'pass' : 'fail'}>
          <span>
            <code className="font-mono text-ink">go test ./...</code>{' '}
            <span className={run.passed ? 'font-semibold text-pass' : 'font-semibold text-fail'}>
              {run.passed ? 'all tests pass' : failures ? `${failures} tests failing` : 'tests failing'}
            </span>
          </span>
          <span className="text-[12px] underline decoration-dotted group-open:hidden">show output</span>
        </Row>
      </summary>
      <pre className="mt-2 ml-5 overflow-x-auto rounded-lg border-2 border-ink bg-ink p-3 font-mono text-[12px] leading-snug text-white/85">
        {run.output}
      </pre>
    </details>
  );
}

export function ToolPart({
  part,
  coder,
  coderName,
  onApproval,
}: {
  part: AnyToolPart;
  coder?: CoderState;
  coderName: string;
  onApproval: (id: string, approved: boolean) => void;
}) {
  const name = getToolName(part);

  if (part.state === 'output-error') {
    return <Row tone="fail">{name} failed: {part.errorText}</Row>;
  }

  // Any tool with ToolApproval set on the server waits here for the user.
  // delegate_to_coding_agent has its own wording below.
  if (part.state === 'approval-requested' && name !== 'delegate_to_coding_agent') {
    return (
      <ApprovalCard
        title={`Run ${name}?`}
        detail="Shipyard wants to run this tool with this input."
        input={JSON.stringify(part.input, null, 2)}
        onDecide={(approved) => onApproval(part.approval.id, approved)}
      />
    );
  }
  if (part.state === 'output-denied' && name !== 'delegate_to_coding_agent') {
    return <Row>You denied {name}.</Row>;
  }

  switch (name) {
    case 'run_tests':
      return <TestsPart part={part} />;
    case 'list_files':
      return <Row>Looked at the project files</Row>;
    case 'read_file': {
      const path = (part.input as { path?: string } | undefined)?.path ?? 'a file';
      return (
        <Row>
          Read <code className="font-mono text-ink">{path}</code>
        </Row>
      );
    }
    case 'delegate_to_coding_agent':
      return <DelegatePart part={part} coder={coder} coderName={coderName} onApproval={onApproval} />;
    default:
      return <Row>{name}</Row>;
  }
}

function DelegatePart({
  part,
  coder,
  coderName,
  onApproval,
}: {
  part: AnyToolPart;
  coder?: CoderState;
  coderName: string;
  onApproval: (id: string, approved: boolean) => void;
}) {
  const task = (part.input as { task?: string } | undefined)?.task;
  const agent = coder?.agent ?? coderName;

  if (part.state === 'input-streaming' || part.state === 'input-available') {
    return <Row tone="busy">Writing a task for {agent}</Row>;
  }

  if (part.state === 'approval-requested') {
    return (
      <ApprovalCard
        title={`Let ${agent} edit the project?`}
        detail="It edits files and runs commands in a sandbox copy of the project. When it finishes, the diff appears here."
        input={task}
        onDecide={(approved) => onApproval(part.approval.id, approved)}
      />
    );
  }

  if (part.state === 'output-denied' || (part.state === 'approval-responded' && !part.approval.approved)) {
    return <Row>You denied the change, so {agent} didn&apos;t run.</Row>;
  }

  if (coder) return <CoderCard state={coder} />;
  return <Row tone="busy">Approved. Starting {agent}…</Row>;
}

/** The Approve / Deny card for a tool call that needs the user's approval. */
function ApprovalCard({
  title,
  detail,
  input,
  onDecide,
}: {
  title: string;
  detail: string;
  input?: string;
  onDecide: (approved: boolean) => void;
}) {
  return (
    <section className="rounded-2xl border-2 border-ink bg-amber p-4 shadow-ink-lg">
      <h3 className="text-[17px] font-bold">{title}</h3>
      <p className="mt-1 text-[14px] text-ink/75">{detail}</p>
      {input && (
        <blockquote className="mt-3 whitespace-pre-wrap rounded-lg border-2 border-ink bg-white px-3 py-2 text-[14px] leading-snug">
          {input}
        </blockquote>
      )}
      <div className="mt-4 flex gap-3">
        <button
          type="button"
          onClick={() => onDecide(true)}
          className="rounded-xl border-2 border-ink bg-ink px-5 py-2 text-[15px] font-bold text-white shadow-[3px_3px_0_0_var(--color-go)] hover:bg-go-deep"
        >
          Approve
        </button>
        <button
          type="button"
          onClick={() => onDecide(false)}
          className="rounded-xl border-2 border-ink bg-white px-5 py-2 text-[15px] font-bold hover:bg-gopher-soft"
        >
          Deny
        </button>
      </div>
    </section>
  );
}
