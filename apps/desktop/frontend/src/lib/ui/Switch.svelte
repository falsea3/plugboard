<script lang="ts">
  let {
    checked = $bindable(false),
    small = false,
    label,
    onchange,
  }: { checked?: boolean; small?: boolean; label?: string; onchange?: (on: boolean) => void } = $props();
</script>

<span class="switch" class:small>
  <input type="checkbox" bind:checked aria-label={label} onchange={() => onchange?.(checked)} />
  <span class="track"></span>
</span>

<style>
  .switch { position: relative; flex: none; width: 34px; height: 20px; }
  .switch.small { width: 30px; height: 18px; margin-top: 1px; }
  .switch input { position: absolute; inset: 0; opacity: 0; margin: 0; }
  .track { position: absolute; inset: 0; border-radius: 999px; background: var(--active); transition: background 0.15s; pointer-events: none; }
  .track::after {
    content: '';
    position: absolute;
    top: 2px;
    left: 2px;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: #fff;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
    transition: transform 0.15s;
  }
  .small .track::after { width: 14px; height: 14px; }
  .switch input:checked + .track { background: var(--accent); }
  .switch input:checked + .track::after { transform: translateX(14px); }
  .small input:checked + .track::after { transform: translateX(12px); }
  .switch input:focus-visible + .track { outline: 2px solid var(--accent); outline-offset: 2px; }
</style>
