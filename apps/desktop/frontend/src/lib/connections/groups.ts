import type { Arrangement, Connection, ConnectionGroup } from '../api/wire';

export type Section = { name: string; collapsed: boolean; items: Connection[] };

export type ConnDrop = { group: string; before: string | null };

export function sections(conns: Connection[], groups: ConnectionGroup[]): Section[] {
  const out: Section[] = groups.filter(g => g.name).map(g => ({ name: g.name, collapsed: g.collapsed, items: [] }));
  const byName = new Map(out.map(s => [s.name, s]));
  const loose: Section = { name: '', collapsed: false, items: [] };
  for (const c of conns) {
    const name = c.group?.trim() ?? '';
    let s = name ? byName.get(name) : loose;
    if (!s) {
      s = { name, collapsed: false, items: [] };
      byName.set(name, s);
      out.push(s);
    }
    s.items.push(c);
  }
  if (loose.items.length > 0 || out.length === 0) out.push(loose);
  return out;
}

export function filterSections(secs: Section[], match: (c: Connection) => boolean): Section[] {
  return secs.map(s => ({ ...s, collapsed: false, items: s.items.filter(match) })).filter(s => s.items.length > 0);
}

export function moveConnection(secs: Section[], id: string, drop: ConnDrop): Section[] {
  const conn = secs.flatMap(s => s.items).find(c => c.id === id);
  if (!conn || drop.before === id) return secs;
  const out = secs.map(s => ({ ...s, items: s.items.filter(c => c.id !== id) }));
  let target = out.find(s => s.name === drop.group);
  if (!target) {
    target = { name: drop.group, collapsed: false, items: [] };
    const loose = out.findIndex(s => !s.name);
    out.splice(drop.group && loose >= 0 ? loose : out.length, 0, target);
  }
  const at = drop.before ? target.items.findIndex(c => c.id === drop.before) : -1;
  target.items.splice(at < 0 ? target.items.length : at, 0, { ...conn, group: drop.group });
  return out;
}

export function moveGroup(secs: Section[], name: string, before: string | null): Section[] {
  const from = secs.findIndex(s => s.name === name);
  if (!name || from < 0 || before === name) return secs;
  const out = secs.slice();
  const [moved] = out.splice(from, 1);
  let at = before === null ? out.findIndex(s => !s.name) : out.findIndex(s => s.name === before);
  if (at < 0) at = out.length;
  out.splice(at, 0, moved);
  return out;
}

export function renameGroup(secs: Section[], from: string, to: string): Section[] {
  return secs.map(s => (s.name === from ? { ...s, name: to, items: s.items.map(c => ({ ...c, group: to })) } : s));
}

export function dropGroup(secs: Section[], name: string): Section[] {
  const gone = secs.find(s => s.name === name);
  if (!name || !gone) return secs;
  const rest = secs.filter(s => s !== gone);
  const loose = rest.find(s => !s.name);
  const moved = gone.items.map(c => ({ ...c, group: '' }));
  if (loose) return rest.map(s => (s === loose ? { ...s, items: [...s.items, ...moved] } : s));
  return [...rest, { name: '', collapsed: false, items: moved }];
}

export function arrangement(secs: Section[]): Arrangement {
  return {
    groups: secs.filter(s => s.name).map(s => ({ name: s.name, collapsed: s.collapsed })),
    connections: secs.flatMap(s => s.items.map(c => ({ id: c.id, group: s.name }))),
  };
}

export function groupNameProblem(name: string, secs: Section[], current = ''): string {
  const n = name.trim();
  if (!n) return 'Enter a name';
  if (n !== current && secs.some(s => s.name === n)) return 'A group with this name exists';
  return '';
}
