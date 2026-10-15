import { getContext, setContext } from 'svelte';

const KEY = Symbol('pane-visible');

export function setPaneVisible(get: () => boolean) {
  setContext(KEY, get);
}

export function paneVisible(): () => boolean {
  return getContext<(() => boolean) | undefined>(KEY) ?? (() => true);
}
