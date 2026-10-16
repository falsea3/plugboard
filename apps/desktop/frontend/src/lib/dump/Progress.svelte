<script lang="ts">
  let { share, label, detail = '' }: { share: number | null; label: string; detail?: string } = $props();
</script>

<div class="progress" role="status" aria-live="polite">
  <div class="bar" class:busy={share === null}>
    <span style:width={share === null ? '' : `${Math.max(2, share * 100)}%`}></span>
  </div>
  <div class="label"><span class="text">{label}</span>{#if share !== null}<span class="pct">{Math.floor(share * 100)}%</span>{/if}</div>
  {#if detail}<div class="detail faint">{detail}</div>{/if}
</div>

<style>
  .progress { display: flex; flex-direction: column; gap: 6px; }
  .bar { position: relative; height: 6px; border-radius: 999px; background: var(--active); overflow: hidden; }
  .bar span { position: absolute; inset: 0 auto 0 0; border-radius: inherit; background: var(--accent); transition: width 0.2s; }
  .bar.busy span { width: 30%; animation: slide 1.1s ease-in-out infinite; }
  @keyframes slide { from { left: -30%; } to { left: 100%; } }
  .label { display: flex; gap: 8px; min-width: 0; font-size: 12.5px; color: var(--text); }
  .text { flex: 1; min-width: 0; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
  .pct { color: var(--text-3); font-variant-numeric: tabular-nums; }
  .detail { font-size: 11.5px; font-variant-numeric: tabular-nums; }
</style>
