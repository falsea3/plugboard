import { describe, expect, it } from 'vitest';
import { parseConnectionUrl } from './url';

describe('parseConnectionUrl', () => {
  it('reads a full PostgreSQL URL', () => {
    const c = parseConnectionUrl('postgres://app:s3cr%40t@db.example.com:6543/shop?sslmode=require');
    expect(c).toMatchObject({
      driver: 'postgres', host: 'db.example.com', port: 6543, user: 'app', password: 's3cr@t',
      database: 'shop', sslMode: 'require', name: 'shop @ db.example.com',
    });
  });

  it('fills defaults and accepts postgresql:// and jdbc:', () => {
    expect(parseConnectionUrl('postgresql://localhost/app')).toMatchObject({ port: 5432, user: 'postgres', database: 'app' });
    expect(parseConnectionUrl('jdbc:postgresql://h:5433/x')).toMatchObject({ host: 'h', port: 5433, database: 'x' });
  });

  it('understands ssl=true and .env lines', () => {
    expect(parseConnectionUrl('DATABASE_URL="postgres://u:p@h/db?ssl=true"')).toMatchObject({ user: 'u', sslMode: 'require' });
  });

  it('reads MySQL and MariaDB URLs with their TLS flags', () => {
    expect(parseConnectionUrl('mysql://root:pw@127.0.0.1/shop?ssl-mode=REQUIRED')).toMatchObject({
      driver: 'mysql', port: 3306, user: 'root', password: 'pw', database: 'shop', sslMode: 'require',
    });
    expect(parseConnectionUrl('mariadb://u@h:3307?tls=false')).toMatchObject({ driver: 'mysql', port: 3307, sslMode: 'disable', database: '' });
    expect(parseConnectionUrl('mysql+pymysql://u@h/d')).toMatchObject({ driver: 'mysql' });
  });

  it('handles IPv6 hosts', () => {
    expect(parseConnectionUrl('postgres://u@[::1]:5432/d').host).toBe('::1');
  });

  it('reads SQLite paths', () => {
    expect(parseConnectionUrl('sqlite:///Users/me/app.db')).toMatchObject({ driver: 'sqlite', file: '/Users/me/app.db', name: 'app' });
    expect(parseConnectionUrl('file:/tmp/x.sqlite3').file).toBe('/tmp/x.sqlite3');
  });

  it('explains what is wrong', () => {
    expect(() => parseConnectionUrl('localhost:5432')).toThrow(/Unsupported scheme/);
    expect(() => parseConnectionUrl('just text')).toThrow(/Expected a URL/);
    expect(() => parseConnectionUrl('mongodb://h/db')).toThrow(/Unsupported scheme/);
  });
});
