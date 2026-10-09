import { SvelteSet } from 'svelte/reactivity';
import { api, type KeyInfo } from '../api/backend';
import { scanPattern } from './folders';

const PAGE = 500;

export class KeyBrowser {
  keys = $state<KeyInfo[]>([]);
  filter = $state('');
  done = $state(true);
  loading = $state(false);
  error = $state('');
  open = new SvelteSet<string>();
  private cursor = '';
  private seq = 0;

  constructor(
    private sessionId: () => string,
    private db: () => string,
  ) {}

  async load() {
    this.seq++;
    this.keys = [];
    this.cursor = '';
    this.done = false;
    this.error = '';
    this.loading = false;
    await this.more();
  }

  async more() {
    if (this.loading || this.done) return;
    const n = this.seq;
    this.loading = true;
    try {
      const page = await api.scanKeys(this.sessionId(), this.db(), scanPattern(this.filter), this.cursor, PAGE);
      if (n !== this.seq) return;
      const known = new Set(this.keys.map(k => k.key));
      this.keys = [...this.keys, ...page.keys.filter(k => !known.has(k.key))];
      this.cursor = page.cursor;
      this.done = page.done;
    } catch (err) {
      if (n !== this.seq) return;
      this.error = err instanceof Error ? err.message : String(err);
      this.done = true;
    } finally {
      if (n === this.seq) this.loading = false;
    }
  }

  search(filter: string) {
    this.filter = filter;
    return this.load();
  }

  added(k: KeyInfo) {
    if (!this.keys.some(x => x.key === k.key)) this.keys = [...this.keys, k];
  }

  removed(key: string) {
    this.keys = this.keys.filter(k => k.key !== key);
  }

  renamed(key: string, to: KeyInfo) {
    this.keys = this.keys.map(k => (k.key === key ? to : k));
  }
}
