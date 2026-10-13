export interface ConfigFile {
  name: string;
  path: string;
}

export interface ConfigText {
  path: string;
  text: string;
  via: 'ssh' | 'sql' | 'live';
  readOnly?: boolean;
}

export interface ConfigProblem {
  line: number;
  message: string;
}

export interface ConfigSaved {
  via: 'ssh' | 'sql' | 'live';
  reloaded: boolean;
  problems: ConfigProblem[];
  restart: string[];
  note: string;
}

export interface Role {
  id: string;
  name: string;
  host: string;
  flags: Record<string, boolean>;
  connLimit: number;
  validUntil: string;
  memberOf: string[];
  grants: string[];
  rules: string;
  kind: string;
  grantable: boolean;
  system: boolean;
  current: boolean;
}

export interface RoleSpec extends Role {
  password: string;
  clearPassword: boolean;
}

export interface RoleFlag {
  key: string;
  label: string;
  hint: string;
  badge: string;
}

export interface RoleForm {
  noun: string;
  flags: RoleFlag[];
  host: boolean;
  validUntil: boolean;
  connLimit: boolean;
  membership: boolean;
  rules: string;
  new: Role;
}

export interface RoleList {
  form: RoleForm;
  roles: Role[];
}
