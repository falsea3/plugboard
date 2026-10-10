import { describe, expect, it } from 'vitest';
import { agentSetup, mcpArgs, shellQuote } from './agents';

const exe = '/Applications/Plugboard.app/Contents/MacOS/plugboard';

describe('agent setup', () => {
  it('adds the flags the user turned on', () => {
    expect(mcpArgs({ write: false, create: false })).toEqual(['mcp']);
    expect(mcpArgs({ write: true, create: true })).toEqual(['mcp', '--write', '--create']);
  });

  it('builds a command for the CLIs and a config for the rest', () => {
    expect(agentSetup('claude', exe, ['mcp']).text).toBe(`claude mcp add plugboard -- ${exe} mcp`);
    expect(agentSetup('codex', exe, ['mcp', '--write']).text).toBe(`codex mcp add plugboard -- ${exe} mcp --write`);
    expect(JSON.parse(agentSetup('cursor', exe, ['mcp']).text)).toEqual({ mcpServers: { plugboard: { command: exe, args: ['mcp'] } } });
    expect(JSON.parse(agentSetup('other', 'C:\\Program Files\\Plugboard\\plugboard.exe', ['mcp']).text).mcpServers.plugboard.command).toBe(
      'C:\\Program Files\\Plugboard\\plugboard.exe',
    );
  });

  it('quotes paths a shell would split', () => {
    expect(shellQuote(exe)).toBe(exe);
    expect(shellQuote('/Users/a b/plugboard')).toBe(`'/Users/a b/plugboard'`);
    expect(shellQuote("/x/it's")).toBe(`'/x/it'\\''s'`);
  });
});
