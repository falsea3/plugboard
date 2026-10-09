import { SQLDialect } from '@codemirror/lang-sql';
import logo from 'simple-icons/icons/apachecassandra.svg?raw';
import type { Engine } from './engine';

const CQL = SQLDialect.define({
  keywords:
    'select from where and in is not null contains key limit per partition allow filtering distinct as json token writetime ttl ' +
    'insert into values if exists update set delete using timestamp begin unlogged logged counter batch apply truncate ' +
    'create alter drop table keyspace type index materialized view function aggregate role user with replication ' +
    'primary clustering order by asc desc static frozen use describe desc list grant revoke on to of',
  types:
    'ascii bigint blob boolean counter date decimal double duration float inet int smallint text time timestamp timeuuid ' +
    'tinyint uuid varchar varint list set map tuple frozen vector',
  doubleQuotedStrings: false,
  slashComments: true,
  identifierQuotes: '"',
});

export const cassandra: Engine = {
  driver: 'cassandra',
  name: 'Cassandra',
  logo,
  background: 'var(--engine-cassandra)',
  file: false,
  fileExtensions: [],
  defaultPort: 9042,
  defaultUser: '',
  defaultDatabase: '',
  schemes: ['cassandra', 'cql'],
  sslModeFromUrl(q) {
    const ssl = (q.get('ssl') ?? q.get('tls') ?? '').toLowerCase();
    return ssl === 'true' || ssl === '1' ? 'verify-full' : undefined;
  },
  canPreferSsl: false,
  databaseHint: 'Optional — keyspace to open',
  userHint: 'Optional — for PasswordAuthenticator',
  sqlDialect: CQL,
};
