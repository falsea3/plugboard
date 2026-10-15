<script lang="ts">
  import Button from './Button.svelte';
  import TextField from './TextField.svelte';
  import SegmentedControl from './SegmentedControl.svelte';
  import Pill from './Pill.svelte';
  import Panel from './Panel.svelte';
  import Switch from './Switch.svelte';
  import ToggleRow from './ToggleRow.svelte';
  import ToggleCard from './ToggleCard.svelte';
  import Select from './Select.svelte';
  import EnvBadge from '../connections/EnvBadge.svelte';

  let view = $state<'data' | 'structure' | 'ddl'>('data');
  let on = $state(true);
  let pick = $state('public');
  let theme = $state<'dark' | 'light'>('dark');

  $effect(() => {
    document.documentElement.dataset.theme = theme;
  });
</script>

<main class="kit">
  <header>
    <h1>Plugboard UI kit</h1>
    <SegmentedControl label="Theme" options={[{ value: 'dark', label: 'Dark' }, { value: 'light', label: 'Light' }]} bind:value={theme} />
  </header>

  <section>
    <h2>Buttons</h2>
    <div class="row">
      <Button variant="primary" kbd="⌘S">Save</Button>
      <Button>New query</Button>
      <Button variant="outline" icon="funnel">Filter</Button>
      <Button variant="ghost">Cancel</Button>
      <Button variant="danger" icon="trash">Delete</Button>
      <Button icon="refresh" label="Reload" variant="outline" />
      <Button variant="primary" disabled>Disabled</Button>
    </div>
    <div class="row">
      <Button size="sm" variant="primary">Apply</Button>
      <Button size="sm">Set NULL</Button>
      <Button size="sm" variant="outline" icon="plus">Row</Button>
      <Button size="sm" variant="ghost" icon="copy" label="Copy" />
    </div>
  </section>

  <section>
    <h2>Fields</h2>
    <div class="grid">
      <TextField icon="search" placeholder="Filter tables" kbd="⇧⌘F" label="Filter tables" />
      <TextField prefix="where" mono placeholder="country = 'UK'" label="WHERE condition" />
      <TextField size="sm" placeholder="Small field" label="Small" />
      <Select value={pick} options={[{ value: 'public', label: 'public' }, { value: 'analytics', label: 'analytics' }]} onchange={v => (pick = v)} aria-label="Schema" />
    </div>
  </section>

  <section>
    <h2>Segmented control</h2>
    <div class="row">
      <SegmentedControl label="View" options={[{ value: 'data', label: 'Data' }, { value: 'structure', label: 'Structure' }, { value: 'ddl', label: 'DDL' }]} bind:value={view} />
      <SegmentedControl size="sm" label="View small" options={[{ value: 'data', label: 'Data' }, { value: 'structure', label: 'Structure' }, { value: 'ddl', label: 'DDL' }]} bind:value={view} />
    </div>
  </section>

  <section>
    <h2>Badges, pills, keys</h2>
    <div class="row">
      <EnvBadge env="local" /><EnvBadge env="dev" /><EnvBadge env="staging" /><EnvBadge env="prod" />
      <Pill>16 rows</Pill><Pill>3 ms</Pill><Pill tone="accent">2 open</Pill><Pill tone="warn">1 change · ⌘S to save</Pill><Pill tone="ok">Connected</Pill><Pill tone="danger">1 error</Pill>
      <span class="kbd">⌘K</span><span class="kbd">⇧⌘↵</span>
    </div>
  </section>

  <section>
    <h2>Switches</h2>
    <div class="row">
      <Switch bind:checked={on} label="Switch" />
      <Switch small checked={false} label="Small switch" />
    </div>
    <div class="grid">
      <ToggleCard bind:checked={on} title="Read only" hint="Writes are refused" />
      <ToggleCard checked={false} title="AI agents (MCP)" hint="Open to MCP clients" />
    </div>
    <div style="max-width: 420px">
      <ToggleRow bind:checked={on}>
        {#snippet title()}Read only{/snippet}
        Writes are refused
      </ToggleRow>
    </div>
  </section>

  <section>
    <h2>Panels</h2>
    <div class="grid">
      <Panel title="Value">
        {#snippet actions()}<Button size="sm" variant="ghost" icon="x" label="Close" />{/snippet}
        <p class="body">A floating panel: the sidebar, the value panel, popovers.</p>
      </Panel>
      <Panel tone="card" title="Result">
        <p class="body">A content card: the grid, the editor, results.</p>
      </Panel>
    </div>
  </section>
</main>

<style>
  .kit { height: 100%; overflow: auto; padding: 32px 40px; background: var(--bg); user-select: text; -webkit-user-select: text; }
  header { display: flex; align-items: center; justify-content: space-between; max-width: 960px; }
  h1 { margin: 0; font-size: 22px; font-weight: 600; letter-spacing: -0.01em; }
  section { max-width: 960px; margin-top: 32px; }
  h2 { margin: 0 0 12px; font-size: 11px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; color: var(--text-3); }
  .row { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; margin-bottom: 12px; }
  .grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
  .body { margin: 0; padding: 14px; color: var(--text-2); }
</style>
