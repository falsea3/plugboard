export type AgentClient = 'claude' | 'codex' | 'cursor' | 'other';

export const CLIENTS: { id: AgentClient; label: string }[] = [
  { id: 'claude', label: 'Claude Code' },
  { id: 'codex', label: 'Codex' },
  { id: 'cursor', label: 'Cursor' },
  { id: 'other', label: 'Other' },
];

export type Setup = { text: string; hint: string };

export function mcpArgs(o: { write: boolean; create: boolean }): string[] {
  return ['mcp', ...(o.write ? ['--write'] : []), ...(o.create ? ['--create'] : [])];
}

export function shellQuote(s: string): string {
  return /^[\w@%+=:,./-]+$/.test(s) ? s : `'${s.replace(/'/g, `'\\''`)}'`;
}

function serversJSON(exe: string, args: string[]): string {
  return JSON.stringify({ mcpServers: { plugboard: { command: exe, args } } }, null, 2);
}

export function agentSetup(client: AgentClient, exe: string, args: string[]): Setup {
  const command = [shellQuote(exe), ...args].join(' ');
  switch (client) {
    case 'claude':
      return { text: `claude mcp add plugboard -- ${command}`, hint: 'Run it in a terminal, then restart Claude Code; /mcp lists Plugboard’s tools.' };
    case 'codex':
      return { text: `codex mcp add plugboard -- ${command}`, hint: 'Run it in a terminal; Codex keeps it in ~/.codex/config.toml and picks it up in the next session.' };
    case 'cursor':
      return { text: serversJSON(exe, args), hint: 'Add this to ~/.cursor/mcp.json (or .cursor/mcp.json in a project), then turn Plugboard on under Settings ▸ MCP.' };
    case 'other':
      return {
        text: serversJSON(exe, args),
        hint: 'Claude Desktop, Windsurf and most other clients take this shape: Plugboard runs as a local stdio server with these arguments. VS Code calls the key “servers” instead of “mcpServers”.',
      };
  }
}
