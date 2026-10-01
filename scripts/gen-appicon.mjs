// Builds every app icon asset from the master SVG in design/appicon.svg.
// npm run icons
import sharp from 'sharp';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const master = readFileSync(join(root, 'design', 'appicon.svg'));

const targets = [
  // Wails turns this into the .icns / .ico at build time.
  { path: join(root, 'apps', 'desktop', 'build', 'appicon.png'), size: 1024, trim: false },
  // In-app logo (home screen, About): the tile without the transparent margin.
  { path: join(root, 'apps', 'desktop', 'frontend', 'src', 'lib', 'assets', 'relay-db-mark.png'), size: 144, trim: true },
];

for (const t of targets) {
  let img = sharp(master, { density: 144 });
  if (t.trim) img = sharp(await img.png().toBuffer()).trim({ threshold: 1 });
  await img.resize(t.size, t.size, { fit: 'contain', background: { r: 0, g: 0, b: 0, alpha: 0 } }).png().toFile(t.path);
  console.log('wrote', t.path);
}
