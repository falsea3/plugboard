<script lang="ts" module>
  import postgres from 'simple-icons/icons/postgresql.svg?raw';
  import mysql from '../assets/mysql-dolphin.svg?raw';
  import sqlite from 'simple-icons/icons/sqlite.svg?raw';
  import type { Driver } from '../wire';

  const LOGOS: Record<Driver, string> = { postgres, mysql, sqlite };
  export const DRIVER_NAMES: Record<Driver, string> = { postgres: 'PostgreSQL', mysql: 'MySQL', sqlite: 'SQLite' };
</script>

<script lang="ts">
  let { driver, size = 32, round = false }: { driver: Driver; size?: number; round?: boolean } = $props();
</script>

<span
  class="mark {driver}"
  class:round
  style:width="{size}px"
  style:height="{size}px"
  title={DRIVER_NAMES[driver]}
  aria-label={DRIVER_NAMES[driver]}
  role="img"
>
  {@html LOGOS[driver]}
</span>

<style>
  .mark {
    flex: none;
    display: grid;
    place-items: center;
    border-radius: 24%;
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.28), inset 0 -1px 0 rgba(0, 0, 0, 0.12);
  }
  .mark.round { border-radius: 50%; }
  .mark :global(svg) {
    width: 62%;
    height: 62%;
    fill: #fff;
    filter: drop-shadow(0 1px 1px rgba(0, 0, 0, 0.22));
  }
  .mark :global(title) { display: none; }
  .postgres { background: linear-gradient(155deg, #6a9bff 0%, #3466e0 55%, #2a52bd 100%); }
  .mysql { background: linear-gradient(155deg, #ffb547 0%, #f28a12 55%, #d96d00 100%); }
  .sqlite { background: linear-gradient(155deg, #5fd3f5 0%, #149bd8 55%, #0b78b8 100%); }
</style>
