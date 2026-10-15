const KEY = 'plugboard.sidebarHidden';

function read(): boolean {
  try {
    return localStorage.getItem(KEY) === '1';
  } catch {
    return false;
  }
}

class Layout {
  sidebarHidden = $state(read());

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
