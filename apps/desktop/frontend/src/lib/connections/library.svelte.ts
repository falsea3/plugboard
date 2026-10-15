import { api, type Connection, type ConnectionGroup } from '../api/backend';
import { arrangement, dropGroup, renameGroup, sections, type Section } from './groups';

type Owner = { connections: Connection[]; notify(err: unknown): void };

export class ConnectionGroups {
  list = $state<ConnectionGroup[]>([]);
  sections = $derived.by(() => sections(this.owner.connections, this.list));

  constructor(private owner: Owner) {}

  async load() {
    this.list = await api.listGroups();
  }

  async apply(secs: Section[]) {
    const arr = arrangement(secs);
    const before = { list: this.list, connections: this.owner.connections };
    const byId = new Map(this.owner.connections.map(c => [c.id, c]));
    this.list = arr.groups;
    this.owner.connections = arr.connections.flatMap(p => {
      const c = byId.get(p.id);
      return c ? [{ ...c, group: p.group }] : [];
    });
    try {
      await api.arrangeConnections(arr);
    } catch (err) {
      this.list = before.list;
      this.owner.connections = before.connections;
      this.owner.notify(err);
    }
  }

  toggle(name: string) {
    return this.apply(this.sections.map(s => (s.name === name ? { ...s, collapsed: !s.collapsed } : s)));
  }

  add(name: string) {
    const secs = this.sections.slice();
    const loose = secs.findIndex(s => !s.name);
    secs.splice(loose < 0 ? secs.length : loose, 0, { name: name.trim(), collapsed: false, items: [] });
    return this.apply(secs);
  }

  rename(from: string, to: string) {
    return this.apply(renameGroup(this.sections, from, to.trim()));
  }

  remove(name: string) {
    return this.apply(dropGroup(this.sections, name));
  }
}
