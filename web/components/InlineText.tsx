/** Renders text with `inline code` spans, the only markdown the agents use here. */
export function InlineText({ text, codeClassName }: { text: string; codeClassName: string }) {
  return (
    <>
      {text.split(/(`[^`\n]+`)/g).map((piece, i) =>
        piece.length > 2 && piece.startsWith('`') && piece.endsWith('`') ? (
          <code key={i} className={codeClassName}>
            {piece.slice(1, -1)}
          </code>
        ) : (
          piece
        ),
      )}
    </>
  );
}
