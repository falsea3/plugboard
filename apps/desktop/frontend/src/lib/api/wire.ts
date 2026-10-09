export type Driver = 'postgres' | 'mysql' | 'sqlite' | 'redis';
export type Env = '' | 'local' | 'dev' | 'staging' | 'prod';

export interface Connection {
  id: string;
  name: string;
  driver: Driver;
  host: string;
  port: number;
  user: string;
  password?: string;
  savePassword: boolean;
  database: string;
  file: string;
  sslMode: string;
  env: Env;
  color: string;
  readOnly: boolean;
  ssh: SSHTunnel;
}

export type SSHAuth = 'password' | 'key' | 'agent';

export interface SSHTunnel {
  enabled: boolean;
  host: string;
  port: number;
  user: string;
  auth: SSHAuth;
  keyFile: string;
  password?: string;
  passphrase?: string;
}

export interface ConnectSecrets {
  password: string;
  sshPassword: string;
  sshPassphrase: string;
}

export interface HostKeyChange {
  host: string;
  fingerprint: string;
}

export interface ConnectResult {
  session?: SessionInfo;
  hostKeyChange?: HostKeyChange;
}

export interface TunnelState {
  sessionId: string;
  state: 'lost' | 'reconnected' | 'failed';
  error?: string;
}

export interface TestResult {
  ok: boolean;
  error?: string;
  serverVersion?: string;
  latencyMs: number;
}

export interface SessionInfo {
  sessionId: string;
  connection: Connection;
  serverVersion: string;
  schemas: string[];
  defaultSchema: string;
  engine: EngineFeatures;
}

export interface EngineFeatures {
  syntax: SqlSyntax;
  booleanType: boolean;
  canUpdateToDefault: boolean;
  canAlterColumns: boolean;
  transactionalDDL: boolean;
  columnTypes: string[];
  keyValue: boolean;
}

export interface SqlSyntax {
  hashComments: boolean;
  dashCommentNeedsSpace: boolean;
  backslashEscapes: boolean;
  escapeStrings: boolean;
  doubleQuotedStrings: boolean;
  dollarQuotes: boolean;
  executableComments: boolean;
}

export interface AppErrorInfo {
  code: string;
  message: string;
  detail?: string;
}


export * from './wireSchema';
export * from './wireKeys';
export * from './wireQuery';
export * from './wireApp';
