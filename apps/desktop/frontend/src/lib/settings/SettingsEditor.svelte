<script lang="ts">
  import { app } from '../app/app.svelte';

  const s = $derived(app.settings);
</script>

<h2>SQL Editor</h2>

<div class="field row">
  <div>
    <div class="label">Font size</div>
    <p class="hint">Applies to the SQL editor.</p>
  </div>
  <div class="stepper">
    <button class="btn icon sm" disabled={s.editorFontSize <= 10} onclick={() => app.updateSettings({ editorFontSize: s.editorFontSize - 1 })} aria-label="Smaller">−</button>
    <span class="mono">{s.editorFontSize}px</span>
    <button class="btn icon sm" disabled={s.editorFontSize >= 24} onclick={() => app.updateSettings({ editorFontSize: s.editorFontSize + 1 })} aria-label="Larger">+</button>
  </div>
</div>
<pre class="sample" style:font-size="{s.editorFontSize}px"><span class="kw">SELECT</span> id, email <span class="kw">FROM</span> customers
<span class="kw">WHERE</span> created_at &gt; <span class="str">'2026-01-01'</span>
<span class="kw">LIMIT</span> <span class="num">50</span>;</pre>

<div class="field">
  <div class="label">Shortcuts</div>
  <dl class="keys">
    <dt><span class="kbd">⌘↵</span></dt><dd>Run the statement under the cursor (or the selection)</dd>
    <dt><span class="kbd">⇧⌘↵</span></dt><dd>Run the whole script</dd>
    <dt><span class="kbd">⌘T</span></dt><dd>New query tab</dd>
    <dt><span class="kbd">⌘K</span></dt><dd>Switch connection</dd>
  </dl>
</div>

<style>
  .stepper { display: flex; align-items: center; gap: 10px; }
  .stepper .mono { min-width: 36px; text-align: center; }
  .sample {
    margin: 0 0 14px;
    padding: 12px 14px;
    border-radius: 8px;
    background: var(--surface);
    border: 1px solid var(--border-subtle);
    font-family: var(--font-mono);
    line-height: 1.6;
    white-space: pre;
    overflow-x: auto;
  }
  .sample .kw { color: var(--syn-kw); font-weight: 500; }
  .sample .str { color: var(--syn-str); }
  .sample .num { color: var(--syn-num); }
  .keys { display: grid; grid-template-columns: 64px 1fr; gap: 8px 12px; margin: 10px 0 0; font-size: 12.5px; }
  .keys dt { text-align: right; }
  .keys dd { margin: 0; color: var(--text-2); }
</style>
