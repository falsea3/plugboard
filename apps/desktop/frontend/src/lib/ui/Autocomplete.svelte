<script lang="ts">
  import { onMount } from 'svelte';

  let {
    value = $bindable(),
    suggestions,
    id,
    autofocus = false,
  }: {
    value: string;
    suggestions: string[];
    id: string;
    autofocus?: boolean;
  } = $props();

  let input = $state<HTMLInputElement>();
  let popup = $state<HTMLDivElement>();
  let suggestionsOpen = $state(false);
  let activeSuggestion = $state(-1);
  let position = $state({ left: 0, top: 0, width: 0, maxHeight: 220, above: false });

  const shownSuggestions = $derived.by(() => {
    const query = value.trim().toLowerCase();
    return suggestions.filter(s => !query || s.toLowerCase().includes(query));
  });

  onMount(() => {
    if (!autofocus) return;
    input?.focus();
    input?.select();
  });

  function updatePosition() {
    if (!input) return;
    const rect = input.getBoundingClientRect();
    const below = window.innerHeight - rect.bottom - 8;
    const above = rect.top - 8;
    const flip = below < Math.min(220, 160) && above > below;
    position = {
      left: rect.left,
      top: flip ? rect.top - 4 : rect.bottom + 4,
      width: rect.width,
      maxHeight: Math.max(0, Math.min(220, flip ? above : below)),
      above: flip,
    };
  }

  function openSuggestions() {
    if (suggestions.length === 0) return;
    updatePosition();
    suggestionsOpen = true;
  }

  function chooseSuggestion(suggestion: string) {
    value = suggestion;
    suggestionsOpen = false;
    activeSuggestion = -1;
  }

  function oninput() {
    openSuggestions();
    activeSuggestion = -1;
  }

  function onwindowpointerdown(e: PointerEvent) {
    const target = e.target as Node | null;
    if (target && (input?.contains(target) || popup?.contains(target))) return;
    suggestionsOpen = false;
    activeSuggestion = -1;
  }

  function toBody(node: HTMLElement) {
    document.body.appendChild(node);
    return { destroy: () => node.remove() };
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      if (shownSuggestions.length === 0) return;
      e.preventDefault();
      updatePosition();
      suggestionsOpen = true;
      const step = e.key === 'ArrowDown' ? 1 : -1;
      activeSuggestion = activeSuggestion < 0
        ? (step > 0 ? 0 : shownSuggestions.length - 1)
        : (activeSuggestion + step + shownSuggestions.length) % shownSuggestions.length;
    } else if (e.key === 'Enter' && suggestionsOpen && activeSuggestion >= 0) {
      e.preventDefault();
      e.stopPropagation();
      chooseSuggestion(shownSuggestions[activeSuggestion]);
    } else if (e.key === 'Escape' && suggestionsOpen) {
      e.preventDefault();
      e.stopPropagation();
      suggestionsOpen = false;
      activeSuggestion = -1;
    }
  }

  function onblur() {
    window.setTimeout(() => {
      suggestionsOpen = false;
      activeSuggestion = -1;
    }, 0);
  }
</script>

<svelte:window onpointerdown={onwindowpointerdown} onscroll={updatePosition} onresize={updatePosition} />

<div class="autocomplete">
  <input
    bind:this={input}
    {id}
    class="input mono"
    bind:value
    spellcheck="false"
    autocomplete="off"
    aria-autocomplete="list"
    aria-controls={suggestionsOpen ? `${id}-suggestions` : undefined}
    aria-expanded={suggestionsOpen && shownSuggestions.length > 0}
    aria-activedescendant={activeSuggestion >= 0 ? `${id}-option-${activeSuggestion}` : undefined}
    oninput={oninput}
    onfocus={openSuggestions}
    onblur={onblur}
    onkeydown={onkeydown}
  />
  {#if suggestionsOpen && shownSuggestions.length > 0}
    <div
      bind:this={popup}
      use:toBody
      id="{id}-suggestions"
      class="suggestions"
      class:above={position.above}
      style:left="{position.left}px"
      style:top="{position.top}px"
      style:width="{position.width}px"
      style:max-height="{position.maxHeight}px"
      role="listbox"
      aria-label="Suggestions"
    >
      {#each shownSuggestions as suggestion, i (suggestion)}
        <button
          id="{id}-option-{i}"
          type="button"
          class="suggestion"
          class:active={i === activeSuggestion}
          role="option"
          aria-selected={i === activeSuggestion}
          tabindex="-1"
          onmousemove={() => (activeSuggestion = i)}
          onmousedown={e => e.preventDefault()}
          onclick={() => chooseSuggestion(suggestion)}
        >{suggestion}</button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .suggestions {
    position: fixed;
    z-index: 60;
    overflow-y: auto;
    padding: 4px;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--elevated);
    box-shadow: var(--shadow-modal);
  }
  .suggestions.above { transform: translateY(-100%); }
  .suggestion {
    display: block;
    width: 100%;
    height: 26px;
    padding: 0 10px;
    border: 0;
    border-radius: 5px;
    background: transparent;
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 12px;
    text-align: left;
  }
  .suggestion:hover, .suggestion.active { background: var(--accent); color: var(--on-accent); }
</style>
