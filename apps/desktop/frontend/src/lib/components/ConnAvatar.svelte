<script lang="ts" module>
  import type { Connection } from '../wire';

  const GRADIENTS = [
    ['#7b6cff', '#4f46e5'],
    ['#ff7a9a', '#e0436b'],
    ['#34d399', '#0f9f74'],
    ['#fbbf24', '#e48a0c'],
    ['#38bdf8', '#0b84d0'],
    ['#c084fc', '#8b3fe0'],
    ['#fb923c', '#e2571d'],
    ['#2dd4bf', '#0e9488'],
  ];

  function hash(s: string): number {
    let h = 2166136261;
    for (let i = 0; i < s.length; i++) h = Math.imul(h ^ s.charCodeAt(i), 16777619);
    return h >>> 0;
  }

  export function initials(name: string): string {
    const words = name.trim().split(/[\s_\-./]+|(?<=[a-z])(?=[A-Z])/).filter(Boolean);
    if (words.length === 0) return '?';
    if (words.length === 1) return [...words[0]].slice(0, 2).join('').toUpperCase();
    return (words[0][0] + words[1][0]).toUpperCase();
  }

  export function gradient(c: Pick<Connection, 'id' | 'name'>): string {
    const [a, b] = GRADIENTS[hash(c.id || c.name) % GRADIENTS.length];
    return `linear-gradient(155deg, ${a}, ${b})`;
  }
</script>

<script lang="ts">
  import DriverMark from './DriverMark.svelte';

  let { connection, size = 32, showDriver = true }: { connection: Connection; size?: number; showDriver?: boolean } = $props();
</script>

<span class="avatar" style:width="{size}px" style:height="{size}px" style:background={gradient(connection)}>
  <span class="letters" style:font-size="{Math.round(size * 0.4)}px">{initials(connection.name)}</span>
  {#if showDriver}
    <span class="driver" style:--d="{Math.max(12, Math.round(size * 0.4))}px">
      <DriverMark driver={connection.driver} size={Math.max(12, Math.round(size * 0.4))} round />
    </span>
  {/if}
</span>

<style>
  .avatar {
    position: relative;
    flex: none;
    display: grid;
    place-items: center;
    border-radius: 24%;
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.25), inset 0 -1px 0 rgba(0, 0, 0, 0.12);
  }
  .letters {
    color: #fff;
    font-weight: 700;
    letter-spacing: -0.02em;
    text-shadow: 0 1px 1px rgba(0, 0, 0, 0.18);
  }
  .driver {
    position: absolute;
    right: calc(var(--d) * -0.3);
    bottom: calc(var(--d) * -0.3);
    display: grid;
    border-radius: 50%;
    box-shadow: 0 0 0 2px var(--avatar-ring, var(--bg));
  }
</style>
