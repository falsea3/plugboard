import logo from 'simple-icons/icons/redis.svg?raw';
import type { Engine } from './engine';

export const redis: Engine = {
  driver: 'redis',
  name: 'Redis',
  logo,
  background: 'var(--engine-redis)',
  file: false,
  fileExtensions: [],
  defaultPort: 6379,
  defaultUser: '',
  defaultDatabase: '0',
  schemes: ['redis', 'rediss', 'valkey', 'valkeys'],
  sslModeFromUrl(q, scheme) {
    if (scheme.endsWith('s')) return q.get('verify') === 'false' ? 'require' : 'verify-full';
    return undefined;
  },
  canPreferSsl: false,
  databaseHint: 'Optional — database number, defaults to 0',
  userHint: 'Optional — for Redis 6+ ACL users',
};
