import { api, AppError, onDumpProgress, onImportProgress } from '../api/backend';
import type { DumpFile, DumpOptions, DumpProgress, DumpResult, ImportOptions, ImportProgress, ImportResult } from '../api/wire';

let seq = 0;

class Job {
  readonly id = `dump-${Date.now()}-${++seq}`;
  running = $state(false);
  stopping = $state(false);
  error = $state<AppError | null>(null);

  stop() {
    if (!this.running) return;
    this.stopping = true;
    api.cancelQuery(this.id).catch(() => {});
  }

  protected async track<T>(listen: () => () => void, work: () => Promise<T>): Promise<T | null> {
    this.running = true;
    this.stopping = false;
    this.error = null;
    const off = listen();
    try {
      return await work();
    } catch (err) {
      this.error = err instanceof AppError ? err : new AppError(String(err));
      return null;
    } finally {
      off();
      this.running = false;
    }
  }
}

export class ExportJob extends Job {
  progress = $state<DumpProgress | null>(null);
  result = $state<DumpResult | null>(null);

  async run(sessionId: string, name: string, opts: DumpOptions) {
    this.progress = null;
    this.result = null;
    const res = await this.track(
      () => onDumpProgress(this.id, p => (this.progress = p)),
      () => api.exportDump(sessionId, this.id, name, opts),
    );
    if (res?.path) this.result = res;
    return res;
  }
}

export class ImportJob extends Job {
  file = $state<DumpFile | null>(null);
  progress = $state<ImportProgress | null>(null);
  result = $state<ImportResult | null>(null);

  async pick() {
    const f = await api.pickDumpFile().catch(err => {
      this.error = err instanceof AppError ? err : new AppError(String(err));
      return null;
    });
    if (f?.path) {
      this.file = f;
      this.result = null;
      this.error = null;
    }
  }

  async run(sessionId: string, opts: ImportOptions) {
    if (!this.file) return null;
    const path = this.file.path;
    this.progress = null;
    this.result = null;
    const res = await this.track(
      () => onImportProgress(this.id, p => (this.progress = p)),
      () => api.importDump(sessionId, this.id, path, opts),
    );
    this.result = res;
    return res;
  }
}
