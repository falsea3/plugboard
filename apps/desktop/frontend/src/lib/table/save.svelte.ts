import { api, type ChangeSet, type ResultColumn } from '../api/backend';
import { app } from '../app/app.svelte';
import type { TableTab, Workspace } from '../app/workspace.svelte';
import { formatCount } from '../ui/format';
import type { TableEdits } from './edits.svelte';
import type { TableRows } from './rows.svelte';
import { failMessage } from './commit';

type Options = {
  ws: Workspace;
  tab: TableTab;
  edits: TableEdits;
  rows: TableRows;
  columns: () => ResultColumn[];
  keyColumns: () => string[];
  showRow: (r: number) => void;
};

export class TableSave {
  saving = $state(false);
  error = $state('');
  preview = $state<{ sql: string[]; confirm: boolean } | null>(null);

  constructor(private o: Options) {}

  private changeSet(): { cs: ChangeSet; rows: number[] } {
    const { changes, rows } = this.o.edits.changes(this.o.columns(), this.o.keyColumns());
    return { cs: { schema: this.o.tab.schema, table: this.o.tab.table, changes }, rows };
  }

  async showPreview(confirm: boolean) {
    try {
      this.preview = { sql: await api.previewChanges(this.o.ws.session.sessionId, this.changeSet().cs), confirm };
    } catch (err) {
      this.error = err instanceof Error ? err.message : String(err);
    }
  }

  commit() {
    if (!this.o.edits.dirty || this.saving) return;
    if (this.o.ws.connection.env === 'prod' && app.settings.confirmProdWrites) void this.showPreview(true);
    else void this.apply();
  }

  async apply() {
    const { edits, rows, tab } = this.o;
    this.preview = null;
    this.saving = true;
    this.error = '';
    const { cs, rows: changed } = this.changeSet();
    try {
      const res = await api.applyChanges(this.o.ws.session.sessionId, cs);
      if (res.error) {
        const failed = res.failedIndex >= 0 ? cs.changes[res.failedIndex] : null;
        edits.failedRow = failed ? changed[res.failedIndex] : null;
        this.error = failMessage(failed?.kind, edits.failedRow, res.error);
        if (edits.failedRow !== null) this.o.showRow(edits.failedRow);
        return;
      }
      app.notify(`Saved ${formatCount(res.applied, 'change')} to ${tab.table}.`, 'info');
      rows.changed();
      await rows.load();
    } catch (err) {
      this.error = err instanceof Error ? err.message : String(err);
    } finally {
      this.saving = false;
    }
  }

  discard() {
    this.o.edits.discard();
    this.error = '';
  }
}
