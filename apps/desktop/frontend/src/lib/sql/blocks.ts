const ROUTINES = new Set(['TRIGGER', 'PROCEDURE', 'FUNCTION', 'EVENT']);
const ENDS_OTHER = new Set(['IF', 'LOOP', 'WHILE', 'REPEAT']);

export class Blocks {
  words = 0;
  private create = false;
  private begin = false;
  private batch = false;
  private routine = false;
  private depth = 0;
  private prev = '';

  word(w: string) {
    const u = w.toUpperCase();
    this.words++;
    if (this.words === 1) {
      this.create = u === 'CREATE';
      this.begin = u === 'BEGIN';
    } else if (this.begin && this.words <= 3 && u === 'BATCH') this.batch = true;
    else if (this.batch) {
      if (this.prev === 'APPLY' && u === 'BATCH') this.batch = this.begin = false;
    } else if (!this.routine) this.routine = this.create && this.words <= 8 && ROUTINES.has(u);
    else if (u === 'END') this.depth--;
    else if (this.prev === 'END' && ENDS_OTHER.has(u)) this.depth++;
    else if (u === 'BEGIN' || (u === 'CASE' && this.prev !== 'END')) this.depth++;
    this.prev = u;
  }

  get open(): boolean {
    return (this.routine && this.depth > 0) || this.batch;
  }
}

export function delimiterAt(s: string, i: number): { delim: string; next: number } | null {
  const m = /^DELIMITER[ \t]+(\S+)[ \t]*(?=\n|$)/i.exec(s.slice(i, i + 64));
  return m ? { delim: m[1], next: i + m[0].length } : null;
}

export const isWordChar = (c: string | undefined) => !!c && /[A-Za-z0-9_]/.test(c);
