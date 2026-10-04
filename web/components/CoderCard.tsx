'use client';

import { useEffect, useRef } from 'react';
import type { CoderEvent, CoderState } from '@/lib/types';
import { InlineText } from './InlineText';

function seconds(ms: number) {
  return `${(ms / 1000).toFixed(ms < 10_000 ? 1 : 0)}s`;
}

function EventLine({ event }: { event: CoderEvent }) {
  switch (event.kind) {
    case 'say':
      return (
        <p className="whitespace-pre-wrap font-sans text-[14px] leading-snug text-white">
          <InlineText text={event.text} codeClassName="font-mono text-[13px] text-gopher" />
        </p>
      );
    case 'run':
      return (
        <div>
          <div className="flex items-baseline gap-2">
            <span className="text-gopher">$</span>
            <span className="text-white">{event.text}</span>
            {event.exit !== undefined && (
              <span className={event.exit === 0 ? 'text-pass' : 'text-[#ff7a7e]'}>
                {event.exit === 0 ? 'ok' : `exit ${event.exit}`}
              </span>
            )}
          </div>
          {/* Show output for test runs and failed commands; other output is noise in a short log. */}
          {event.output && (/\bgo test\b/.test(event.text) || (event.exit ?? 0) !== 0) && (
            <pre className="mt-1 ml-4 whitespace-pre-wrap text-[12px] leading-snug text-white/55">{event.output}</pre>
          )}
        </div>
      );
    case 'edit':
      return (
        <div className="text-go">
          <span aria-hidden>✎ </span>edited {event.text}
        </div>
      );
    case 'read':
      return <div className="text-white/50">read {event.text}</div>;
    default:
      return <div className="text-white/50">{event.text}</div>;
  }
}

function Diff({ diff }: { diff: string }) {
  return (
    <pre className="overflow-x-auto rounded-lg border-2 border-ink bg-white p-3 font-mono text-[12.5px] leading-[1.45]">
      {diff.split('\n').map((line, i) => {
        const tone = line.startsWith('+++') || line.startsWith('---')
          ? 'font-bold text-ink'
          : line.startsWith('+')
            ? 'bg-pass/15 text-[#0d6b40]'
            : line.startsWith('-')
              ? 'bg-fail/12 text-[#a3242a]'
              : line.startsWith('@@')
                ? 'text-go-deep'
                : 'text-muted';
        return (
          <div key={i} className={`${tone} -mx-3 px-3`}>
            {line || ' '}
          </div>
        );
      })}
    </pre>
  );
}

export function CoderCard({ state }: { state: CoderState }) {
  const events = state.events ?? [];
  const logRef = useRef<HTMLDivElement>(null);
  const working = state.status === 'starting' || state.status === 'running';

  useEffect(() => {
    const el = logRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [events.length, events.at(-1)?.text]);

  return (
    <section className="overflow-hidden rounded-2xl border-2 border-ink bg-white shadow-ink-lg">
      <header className="flex items-center justify-between gap-3 border-b-2 border-ink bg-gopher-soft px-4 py-2.5">
        <div className="flex items-center gap-2.5">
          <span
            aria-hidden
            className={[
              'size-3 rounded-full border-2 border-ink',
              working ? 'animate-pulse bg-go' : state.status === 'done' ? 'bg-pass' : 'bg-fail',
            ].join(' ')}
          />
          <h3 className="text-[15px] font-bold">
            {state.status === 'starting' && `Starting ${state.agent}`}
            {state.status === 'running' && `${state.agent} is working`}
            {state.status === 'done' && `${state.agent} finished`}
            {state.status === 'error' && `${state.agent} failed`}
          </h3>
        </div>
        <span className="font-mono text-[13px] tabular-nums text-muted">{seconds(state.elapsedMs)}</span>
      </header>

      <div
        ref={logRef}
        className="terminal max-h-72 space-y-2 overflow-y-auto bg-ink px-4 py-3 font-mono text-[13px] leading-snug"
        aria-live="polite"
      >
        {events.length === 0 && (
          <p className="text-white/50">Copying the workspace into a sandbox and starting {state.agent}…</p>
        )}
        {events.map((e) => (
          <EventLine key={e.id} event={e} />
        ))}
      </div>

      {state.status === 'error' && state.error && (
        <p className="border-t-2 border-ink bg-fail/10 px-4 py-3 text-[14px] text-[#a3242a]">{state.error}</p>
      )}

      {state.status === 'done' && (
        <div className="space-y-2 border-t-2 border-ink px-4 py-3">
          <p className="text-[14px] font-semibold">
            {state.filesChanged?.length
              ? `Changed ${state.filesChanged.join(', ')}`
              : 'No files changed'}
          </p>
          {state.diff && <Diff diff={state.diff} />}
        </div>
      )}
    </section>
  );
}
