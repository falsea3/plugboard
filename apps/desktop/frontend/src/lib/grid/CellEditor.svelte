<script lang="ts">
  import Select from '../ui/Select.svelte';

  let {
    text = $bindable(),
    options,
    placeholder,
    width,
    label,
    oninput,
    onkey,
    onblur,
    onpick,
    onclose,
  }: {
    text: string;
    options: string[] | null;
    placeholder: string;
    width: number;
    label: string;
    oninput: () => void;
    onkey: (e: KeyboardEvent) => void;
    onblur: () => void;
    onpick: (value: string) => void;
    onclose: (picked: boolean, key?: KeyboardEvent) => void;
  } = $props();

  const multi = $derived(text.includes('\n') || text.length > 60);

  function focusEditor(node: HTMLTextAreaElement) {
    node.focus();
    node.setSelectionRange(node.value.length, node.value.length);
  }
</script>

{#if options}
  <div class="cell-editor list">
    <Select value={text} options={options.map(o => ({ value: o, label: o }))} {placeholder} startOpen onchange={onpick} {onclose} aria-label={label} />
  </div>
{:else}
  <textarea
    class="cell-editor"
    class:multi
    style:min-width="{width}px"
    bind:value={text}
    {oninput}
    onkeydown={onkey}
    {onblur}
    {placeholder}
    spellcheck="false"
    use:focusEditor
  ></textarea>
{/if}

<style>
  .cell-editor.list { padding: 0; box-shadow: none; background: none; overflow: visible; white-space: normal; }
  .cell-editor.list :global(.select-button) { height: 28px; border-radius: 3px; }
  .cell-editor {
    position: absolute;
    top: -1px;
    left: -1px;
    width: calc(100% + 2px);
    height: 28px;
    margin: 0;
    padding: 4px 8px;
    border: 0;
    border-radius: 3px;
    outline: none;
    box-shadow: 0 0 0 2px var(--accent), 0 6px 20px rgba(0, 0, 0, 0.25);
    background: var(--bg);
    color: var(--text);
    font: inherit;
    line-height: 20px;
    resize: none;
    overflow: hidden;
    white-space: pre;
  }
  .cell-editor.multi {
    width: max(calc(100% + 2px), 340px);
    height: 132px;
    white-space: pre-wrap;
    overflow: auto;
  }
  .cell-editor::placeholder { color: var(--text-3); font-style: italic; }
</style>
