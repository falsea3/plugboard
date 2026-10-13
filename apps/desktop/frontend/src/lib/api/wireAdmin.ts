export interface ConfigFile {
  name: string;
  path: string;
}

export interface ConfigText {
  path: string;
  text: string;
  via: 'ssh' | 'sql';
}

export interface ConfigProblem {
  line: number;
  message: string;
}

export interface ConfigSaved {
  via: 'ssh' | 'sql';
  reloaded: boolean;
  problems: ConfigProblem[];
  restart: string[];
}

export interface Role {
  name: string;
  login: boolean;
  superuser: boolean;
  createDB: boolean;
  createRole: boolean;
  inherit: boolean;
  replication: boolean;
  bypassRLS: boolean;
  connLimit: number;
  validUntil: string;
  memberOf: string[];
  system: boolean;
  current: boolean;
}

export interface RoleSpec extends Role {
  password: string;
  clearPassword: boolean;
}
