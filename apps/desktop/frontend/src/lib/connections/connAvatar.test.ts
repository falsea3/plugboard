import { describe, expect, it } from 'vitest';
import { gradient, initials } from './ConnAvatar.svelte';

describe('ConnAvatar', () => {
  it('takes initials from the connection name', () => {
    expect(initials('Shop via SSH')).toBe('SV');
    expect(initials('Redis (sample)')).toBe('RS');
    expect(initials('(db)')).toBe('DB');
    expect(initials('pl')).toBe('PL');
    expect(initials('music_library')).toBe('ML');
    expect(initials('prodReplica')).toBe('PR');
    expect(initials('  ')).toBe('?');
    expect(initials('Ёлка')).toBe('ЁЛ');
  });

  it('keeps a stable colour per connection', () => {
    expect(gradient({ id: 'abc', name: 'x' })).toBe(gradient({ id: 'abc', name: 'renamed' }));
  });
});
