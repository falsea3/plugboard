import type { Page } from '@playwright/test';

export async function installImport(page: Page) {
  await page.addInitScript(() => {
    const w = window as any;
    const App = w.go.api.App;
    const conn = (name: string, driver: string, host: string, port: number, group = '') => ({
      id: '', name, driver, host, port, user: 'app', savePassword: true, database: 'main', file: '', sslMode: '', env: 'dev', group, color: '', readOnly: false, aiAccess: false,
      ssh: { enabled: false, host: '', port: 22, user: '', auth: 'password', keyFile: '' },
    });
    const scan = {
      root: '/Users/demo/Library/DBeaverData/workspace6',
      candidates: [
        { key: 'General/pg-1', project: 'General', connection: conn('Billing', 'postgres', 'billing.internal', 5432, 'Team'), note: '', skip: '', exists: false, password: true },
        { key: 'General/ch-2', project: 'General', connection: conn('Clicks', 'clickhouse', 'olap.internal', 9000), note: 'port 8123 (HTTP) becomes 9000, ClickHouse’s native port', skip: '', exists: false, password: false },
        { key: 'General/pg-3', project: 'General', connection: conn('Shop', 'postgres', 'db.internal', 5432), note: '', skip: '', exists: true, password: true },
        { key: 'General/ora-4', project: 'General', connection: conn('Ledger', 'postgres', 'ora', 1521), note: '', skip: 'Plugboard doesn’t connect to oracle databases', exists: false, password: false },
      ],
    };
    App.FindDBeaver = () => Promise.resolve(scan);
    App.ChooseDBeaverFolder = () => Promise.resolve({ root: '', candidates: [] });
    App.ImportDBeaver = (root: string, keys: string[]) => {
      w.lastImportKeys = { root, keys };
      return Promise.resolve({ added: keys.length, failed: [] });
    };
  });
}
