import { SQLDialect } from '@codemirror/lang-sql';
import logo from 'simple-icons/icons/clickhouse.svg?raw';
import type { Engine } from './engine';

const ClickHouseSQL = SQLDialect.define({
  keywords:
    'select from where and or not in is null like ilike between as on join inner left right full cross any all asof array global local ' +
    'group by order having limit offset with totals rollup cube settings format final sample prewhere union except intersect distinct ' +
    'case when then else end if exists insert into values create table view materialized dictionary database function engine ' +
    'partition primary key ttl alter add drop modify column rename to optimize truncate show describe desc use set system kill query ' +
    'explain ast syntax pipeline asc desc nulls first last interval',
  types:
    'uint8 uint16 uint32 uint64 uint128 uint256 int8 int16 int32 int64 int128 int256 float32 float64 bfloat16 decimal decimal32 decimal64 ' +
    'string fixedstring uuid date date32 datetime datetime64 enum8 enum16 array tuple map nullable lowcardinality bool ipv4 ipv6 json',
  backslashEscapes: true,
  hashComments: true,
  identifierQuotes: '`"',
});

export const clickhouse: Engine = {
  driver: 'clickhouse',
  name: 'ClickHouse',
  logo,
  background: 'var(--engine-clickhouse)',
  ink: 'var(--engine-clickhouse-ink)',
  file: false,
  fileExtensions: [],
  defaultPort: 9000,
  defaultUser: 'default',
  defaultDatabase: 'default',
  schemes: ['clickhouse', 'ch'],
  sslModeFromUrl(q) {
    const secure = (q.get('secure') ?? '').toLowerCase();
    if (secure === 'true' || secure === '1') return q.get('skip_verify') === 'true' ? 'require' : 'verify-full';
    return undefined;
  },
  canPreferSsl: false,
  databaseHint: 'Optional — defaults to the user’s database',
  userHint: '',
  sqlDialect: ClickHouseSQL,
};
