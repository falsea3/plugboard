// Renders candidate Relay DB app icons side by side for review.
// node scripts/icon-variants.mjs  →  design/icon-variants/*.{svg,png} + sheet.png
// Promote one with: cp design/icon-variants/<name>.svg design/appicon.svg && npm run icons
import sharp from 'sharp';
import { mkdirSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const out = join(root, 'design', 'icon-variants');
mkdirSync(out, { recursive: true });

// Same tile, stroke weight and two-tone scheme as the Relay icon.
const W = 84;
const tile = `
  <defs>
    <linearGradient id="bg" x1="0" y1="100" x2="0" y2="924" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="#fcfcfe"/><stop offset="1" stop-color="#e3e5ef"/>
    </linearGradient>
  </defs>
  <rect x="102" y="102" width="820" height="820" rx="184" fill="url(#bg)" stroke="#dcdde6" stroke-width="4"/>`;
const svg = body => `<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">${tile}${body}</svg>`;
const line = (d, color, w = W) => `<path d="${d}" fill="none" stroke="${color}" stroke-width="${w}" stroke-linecap="round" stroke-linejoin="round"/>`;

// Relay indigo, and a teal pair for the database side of the family.
const indigo = '#5b67f2', indigoLight = '#9ba4ff';
const teal = '#0fa38e', tealLight = '#7ddcc9';

// Picked direction: a solid database cylinder carrying the Relay chevrons,
// in Relay's own indigo so the product reads as part of the family.
const cylinder = (id, top, bottom, lid, chevron, chevronLight) => svg(`
    <defs>
      <linearGradient id="${id}" x1="0" y1="230" x2="0" y2="800" gradientUnits="userSpaceOnUse">
        <stop offset="0" stop-color="${top}"/><stop offset="1" stop-color="${bottom}"/>
      </linearGradient>
    </defs>
    <path d="M252 312 C252 222 772 222 772 312 V712 C772 802 252 802 252 712 Z" fill="url(#${id})"/>
    <ellipse cx="512" cy="312" rx="260" ry="90" fill="${lid}"/>
    <path d="M252 312 C252 402 772 402 772 312" fill="none" stroke="#ffffff" stroke-opacity="0.3" stroke-width="10"/>
    ${line('M388 470 L500 548 L388 626', chevron, 64)}
    ${line('M636 560 L524 638 L636 716', chevronLight, 64)}`);

const variants = {
  'e1-indigo': cylinder('e1', '#6b77ff', '#4a55e0', '#a9b0ff', '#ffffff', '#c9cdff'),
  'e2-indigo-deep': cylinder('e2', '#5b67f2', '#3c46c4', '#9ba4ff', '#ffffff', '#b8beff'),
  'e3-indigo-soft': cylinder('e3', '#8a93ff', '#5b67f2', '#c4c9ff', '#ffffff', '#dfe2ff'),
};

const tiles = [];
for (const [name, s] of Object.entries(variants)) {
  const file = join(out, `${name}.png`);
  writeFileSync(join(out, `${name}.svg`), s);
  await sharp(Buffer.from(s)).png().toFile(file);
  tiles.push({ name, png: await sharp(Buffer.from(s)).resize(256, 256).png().toBuffer(), small: await sharp(Buffer.from(s)).resize(32, 32).png().toBuffer() });
  console.log('wrote', file);
}

// Contact sheet: 256px plus a 32px (Dock-small) preview under each, with Relay for reference.
const relay = await sharp(join(root, 'apps', 'desktop', 'build', 'appicon.png')).resize(256, 256).png().toBuffer();
const all = [{ name: 'current icon', png: relay, small: await sharp(relay).resize(32, 32).png().toBuffer() }, ...tiles];
const cell = 300;
const sheet = sharp({ create: { width: cell * all.length, height: 380, channels: 4, background: '#2a2a2e' } });
const labels = all.map((t, i) => `<text x="${i * cell + cell / 2}" y="365" fill="#ccc" font-family="Helvetica" font-size="18" text-anchor="middle">${t.name}</text>`).join('');
await sheet
  .composite([
    ...all.flatMap((t, i) => [
      { input: t.png, left: i * cell + 22, top: 16 },
      { input: t.small, left: i * cell + cell / 2 - 16, top: 290 },
    ]),
    { input: Buffer.from(`<svg xmlns="http://www.w3.org/2000/svg" width="${cell * all.length}" height="380">${labels}</svg>`), left: 0, top: 0 },
  ])
  .png()
  .toFile(join(out, 'sheet.png'));
console.log('wrote sheet');
