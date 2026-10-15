import { describe, expect, it } from 'vitest';
import type { Connection } from '../api/wire';
import { connectionUrl, inScope, matches, nextId, sshRoute } from './home';

const pg = {
  id: 'a', name: 'HB Stage', driver: 'postgres', host: 'db.internal', port: 5432, user: 'app', database: 'shop', file: '', env: 'staging', group: 'HB',
  ssh: { enabled: true, host: 'bastion', port: 22, user: 'deploy' },
} as Connection;
const lite = { id: 'b', name: 'Music', driver: 'sqlite', file: '/x/music.db', host: '', port: 0, user: '', database: '', env: 'local', ssh: { enabled: true } } as Connection;

describe('home', () => {
  it('builds a connection URL and SSH route', () => {
    expect(connectionUrl(pg)).toEqual({ scheme: 'postgres://', address: 'app@db.internal:5432', database: '/shop' });
    expect(connectionUrl(lite)).toEqual({ scheme: '', address: '/x/music.db', database: '' });
    expect(sshRoute(pg)).toBe('deploy@bastion');
    expect(sshRoute({ ...pg, ssh: { ...pg.ssh, host: '', port: 2222 } })).toBe('deploy@db.internal:2222');
    expect(sshRoute(lite)).toBe('');
  });

  it('filters by scope and text', () => {
    expect(inScope({ kind: 'group', name: 'HB' }, pg)).toBe(true);
    expect(inScope({ kind: 'group', name: '' }, lite)).toBe(true);
    expect(inScope({ kind: 'engine', driver: 'mysql' }, pg)).toBe(false);
    expect(matches('postgre', pg)).toBe(true);
    expect(matches(' hb ', pg)).toBe(true);
    expect(matches('redis', pg)).toBe(false);
  });

  it('steps through ids and stops at the ends', () => {
    expect(nextId(['a', 'b'], 'a', 1)).toBe('b');
    expect(nextId(['a', 'b'], 'b', 1)).toBe('b');
    expect(nextId(['a', 'b'], 'x', -1)).toBe('b');
    expect(nextId([], 'x', 1)).toBe('');
  });
});
