// Release notes are the changelog's Markdown, rendered as Svelte elements
// rather than HTML: latest.json isn't signed, so its text must never become
// markup in a window that can call the backend.

export type Span = { text: string; strong?: boolean; code?: boolean };
export type Block = { kind: 'heading' | 'item' | 'text'; spans: Span[] };

/** The few things the changelog uses: ### headings, - items, **bold** and `code`. */
export function parseNotes(markdown: string): Block[] {
  const blocks: Block[] = [];
  for (const raw of markdown.split('\n')) {
    const line = raw.trim();
    if (!line) continue;
    const heading = /^#{1,6}\s+(.*)$/.exec(line);
    const item = /^[-*]\s+(.*)$/.exec(line);
    if (heading) blocks.push({ kind: 'heading', spans: parseSpans(heading[1]) });
    else if (item) blocks.push({ kind: 'item', spans: parseSpans(item[1]) });
    else blocks.push({ kind: 'text', spans: parseSpans(line) });
  }
  return blocks;
}

function parseSpans(text: string): Span[] {
  const spans: Span[] = [];
  const re = /\*\*(.+?)\*\*|`([^`]+)`|\[([^\]]+)\]\([^)]*\)/g;
  let last = 0;
  for (let m = re.exec(text); m; m = re.exec(text)) {
    if (m.index > last) spans.push({ text: text.slice(last, m.index) });
    if (m[1] !== undefined) spans.push({ text: m[1], strong: true });
    else if (m[2] !== undefined) spans.push({ text: m[2], code: true });
    else spans.push({ text: m[3] }); // a link keeps its words; the notes don't open URLs
    last = re.lastIndex;
  }
  if (last < text.length) spans.push({ text: text.slice(last) });
  return spans;
}
