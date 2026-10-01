import type {} from './wire'; // window.runtime

export async function copyToClipboard(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    // WKWebView can refuse the async clipboard API; the Wails runtime can't.
    await window.runtime?.ClipboardSetText?.(text);
  }
}
