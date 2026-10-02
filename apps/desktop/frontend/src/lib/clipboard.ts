import type {} from './wire';

export async function copyToClipboard(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    await window.runtime?.ClipboardSetText?.(text);
  }
}
