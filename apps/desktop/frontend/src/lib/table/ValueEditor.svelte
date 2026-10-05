<script lang="ts" module>
  import type { CellValue } from '../api/wire';
</script>

<script lang="ts">
  import { jsonToSave } from '../json/text';
  import JsonEditor from '../json/JsonEditor.svelte';
  import TextEditor from '../ui/TextEditor.svelte';

  export type ValueEdit = { r: number; c: number; title: string; value: CellValue; json: boolean; readOnly: boolean };

  let { edit, onsave, onclose }: { edit: ValueEdit; onsave: (value: string) => void; onclose: () => void } = $props();

  const text = $derived(edit.value === null ? '' : String(edit.value));

  function save(next: string) {
    if (edit.json) {
      const value = jsonToSave(text, next);
      if (value === null) onclose();
      else onsave(value);
    } else if (next === text && edit.value !== null) {
      onclose();
    } else {
      onsave(next);
    }
  }
</script>

{#if edit.json}
  <JsonEditor title={edit.title} value={text} readOnly={edit.readOnly} onsave={save} {onclose} />
{:else}
  <TextEditor title={edit.title} value={text} isNull={edit.value === null} readOnly={edit.readOnly} onsave={save} {onclose} />
{/if}
