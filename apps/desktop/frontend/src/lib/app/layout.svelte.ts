const KEY = 'plugboard.sidebarHidden';

function read(): boolean {
  try {
    return localStorage.getItem(KEY) === '1';
  } catch {
    return false;
  }
}

function isFullscreen(): boolean {
  return typeof window !== 'undefined' && window.innerWidth >= screen.width && window.innerHeight >= screen.height;
}

const isMac = typeof navigator !== 'undefined' && /Mac/.test(navigator.userAgent);

class Layout {
  sidebarHidden = $state(read());
  fullscreen = $state(isFullscreen());
  windowButtons = $derived(isMac && !this.fullscreen);

  constructor() {
    if (typeof window !== 'undefined') window.addEventListener('resize', () => (this.fullscreen = isFullscreen()));
  }

  toggleSidebar() {
    this.sidebarHidden = !this.sidebarHidden;
    try {
      localStorage.setItem(KEY, this.sidebarHidden ? '1' : '0');
    } catch {
      return;
    }
  }
}

export const layout = new Layout();
