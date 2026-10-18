import { api } from '../api/backend';

class Pictures {
  byId = $state<Record<string, string>>({});

  async load() {
    this.byId = await api.connectionImages().catch(() => this.byId);
  }

  async set(id: string, url: string) {
    await api.setConnectionImage(id, url);
    if (url) this.byId[id] = url;
    else delete this.byId[id];
  }
}

export const pictures = new Pictures();

export async function squarePicture(file: File, size = 128): Promise<string> {
  const src = URL.createObjectURL(file);
  try {
    const img = new Image();
    img.src = src;
    await img.decode();
    const w = img.naturalWidth || size;
    const h = img.naturalHeight || size;
    const side = Math.min(w, h);
    const canvas = document.createElement('canvas');
    canvas.width = canvas.height = size;
    const ctx = canvas.getContext('2d');
    if (!ctx) throw new Error('This picture can’t be drawn.');
    ctx.imageSmoothingQuality = 'high';
    ctx.drawImage(img, (w - side) / 2, (h - side) / 2, side, side, 0, 0, size, size);
    return canvas.toDataURL('image/png');
  } catch {
    throw new Error(`${file.name} isn’t a picture Plugboard can read.`);
  } finally {
    URL.revokeObjectURL(src);
  }
}
