export interface KeyInfo {
  name: string;
  key: string;
  type: string;
}

export interface KeyPage {
  keys: KeyInfo[];
  cursor: string;
  done: boolean;
}

export interface KeyItem {
  field: string;
  label?: string;
  value: string;
  score?: string;
  binary?: boolean;
}

export interface KeyValue {
  key: string;
  type: string;
  ttl: number;
  size: number;
  text: string;
  binary?: boolean;
  truncated?: boolean;
  items: KeyItem[];
  cursor: string;
  done: boolean;
}

export interface KeyEdit {
  key: string;
  op: string;
  field: string;
  value: string;
  score: string;
  old: string;
  index: number;
}
