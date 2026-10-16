import type { Connection } from './wire';

export interface ImportCandidate {
  key: string;
  project: string;
  connection: Connection;
  note: string;
  skip: string;
  exists: boolean;
  password: boolean;
}

export interface ImportScan {
  root: string;
  candidates: ImportCandidate[];
}

export interface ImportDone {
  added: number;
  failed: string[];
}
