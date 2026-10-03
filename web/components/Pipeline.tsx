export type Phase = 'idle' | 'sending' | 'thinking' | 'approval' | 'coding';

type Node = { title: string; detail: string };

/**
 * The path a message takes, lit up as it happens:
 * browser → Go endpoint → go-ai agent → coding agent.
 */
export function Pipeline({
  phase,
  chatModel,
  coder,
}: {
  phase: Phase;
  chatModel: string;
  coder: string;
}) {
  const nodes: Node[] = [
    { title: 'useChat', detail: 'React, in your browser' },
    { title: 'POST /api/chat', detail: 'Go net/http server' },
    { title: 'go-ai agent', detail: chatModel },
    { title: coder, detail: 'local sandbox' },
  ];

  // Which nodes are lit and which links carry data, per phase.
  const lit: Record<Phase, number[]> = {
    idle: [],
    sending: [0, 1],
    thinking: [1, 2],
    approval: [0, 2],
    coding: [2, 3],
  };
  const activeLink: Record<Phase, number | null> = {
    idle: null,
    sending: 0,
    thinking: 1,
    approval: null,
    coding: 2,
  };

  return (
    <ol className="flex items-center" aria-label="Request pipeline">
      {nodes.map((node, i) => {
        const on = lit[phase].includes(i);
        const waiting = phase === 'approval' && i === 0;
        return (
          <li key={node.title} className="flex items-center">
            <div
              className={[
                'rounded-xl border-2 border-ink px-3 py-1.5 transition-colors duration-200',
                waiting ? 'bg-amber shadow-ink' : on ? 'bg-go text-white shadow-ink' : 'bg-white',
              ].join(' ')}
            >
              <div className="font-mono text-[13px] font-bold leading-tight whitespace-nowrap">{node.title}</div>
              <div
                className={[
                  'text-[11px] leading-tight whitespace-nowrap',
                  on && !waiting ? 'text-white/85' : 'text-muted',
                ].join(' ')}
              >
                {waiting ? 'waiting for you' : node.detail}
              </div>
            </div>
            {i < nodes.length - 1 && (
              <div
                aria-hidden
                className={[
                  'mx-1.5 h-[3px] w-10 lg:w-16',
                  activeLink[phase] === i
                    ? 'link-active'
                    : phase === 'approval' && i === 2
                      ? 'link-waiting'
                      : 'bg-ink/20',
                ].join(' ')}
              />
            )}
          </li>
        );
      })}
    </ol>
  );
}
