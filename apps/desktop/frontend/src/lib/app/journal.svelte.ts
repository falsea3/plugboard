export type Entry = { id: number; at: Date; level: 'error' | 'info'; text: string; detail: string };

const LIMIT = 500;
let seq = 0;

class Journal {
  entries = $state<Entry[]>([]);
  open = $state(false);
  unseen = $state(0);

  add(level: Entry['level'], text: string, detail = '') {
    this.entries.push({ id: ++seq, at: new Date(), level, text, detail });
    if (this.entries.length > LIMIT) this.entries.splice(0, this.entries.length - LIMIT);
    if (!this.open && level === 'error') this.unseen++;
  }

  show() {
    this.open = true;
    this.unseen = 0;
  }

  toggle() {
    if (this.open) this.open = false;
    else this.show();
  }

  clear() {
    this.entries = [];
    this.unseen = 0;
  }
}

export const journal = new Journal();
