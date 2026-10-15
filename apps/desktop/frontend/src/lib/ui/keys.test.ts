import { describe, expect, it } from 'vitest';
import { pcKeys } from './keys';

describe('pcKeys', () => {
  it('spells shortcuts the Windows and Linux way', () => {
    expect(pcKeys('⇧⌘F')).toBe('Ctrl+Shift+F');
    expect(pcKeys('⌘↵')).toBe('Ctrl+Enter');
    expect(pcKeys('⌥⌫')).toBe('Alt+Backspace');
    expect(pcKeys('⌃Tab')).toBe('Ctrl+Tab');
    expect(pcKeys('Settings (⌘,)')).toBe('Settings (Ctrl+,)');
    expect(pcKeys('Hide the sidebar (⌘\\)')).toBe('Hide the sidebar (Ctrl+\\)');
    expect(pcKeys('⇧↵')).toBe('Shift+Enter');
    expect(pcKeys('1 change · ⌘S to save')).toBe('1 change · Ctrl+S to save');
    expect(pcKeys('Write SQL… ⌘↵ runs, ⇧⌘↵ runs everything')).toBe('Write SQL… Ctrl+Enter runs, Ctrl+Shift+Enter runs everything');
    expect(pcKeys('no shortcut')).toBe('no shortcut');
    expect(pcKeys('↵')).toBe('Enter');
  });
});
