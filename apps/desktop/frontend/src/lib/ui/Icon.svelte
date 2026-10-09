<script lang="ts" module>
  import plus from '@phosphor-icons/core/regular/plus.svg?raw';
  import download from '@phosphor-icons/core/regular/download-simple.svg?raw';
  import arrowCircleUp from '@phosphor-icons/core/regular/arrow-circle-up.svg?raw';
  import x from '@phosphor-icons/core/regular/x.svg?raw';
  import search from '@phosphor-icons/core/regular/magnifying-glass.svg?raw';
  import table from '@phosphor-icons/core/regular/table.svg?raw';
  import view from '@phosphor-icons/core/regular/eye.svg?raw';
  import refresh from '@phosphor-icons/core/regular/arrow-clockwise.svg?raw';
  import play from '@phosphor-icons/core/fill/play-fill.svg?raw';
  import stop from '@phosphor-icons/core/fill/stop-fill.svg?raw';
  import chevronLeft from '@phosphor-icons/core/regular/caret-left.svg?raw';
  import chevronRight from '@phosphor-icons/core/regular/caret-right.svg?raw';
  import chevronDown from '@phosphor-icons/core/regular/caret-down.svg?raw';
  import chevronsLeft from '@phosphor-icons/core/regular/caret-double-left.svg?raw';
  import chevronsRight from '@phosphor-icons/core/regular/caret-double-right.svg?raw';
  import database from '@phosphor-icons/core/regular/database.svg?raw';
  import unplug from '@phosphor-icons/core/regular/plugs.svg?raw';
  import trash from '@phosphor-icons/core/regular/trash.svg?raw';
  import pencil from '@phosphor-icons/core/regular/pencil-simple.svg?raw';
  import code from '@phosphor-icons/core/regular/file-sql.svg?raw';
  import key from '@phosphor-icons/core/fill/key-fill.svg?raw';
  import columns from '@phosphor-icons/core/regular/columns.svg?raw';
  import rows from '@phosphor-icons/core/regular/rows.svg?raw';
  import folder from '@phosphor-icons/core/regular/folder-open.svg?raw';
  import check from '@phosphor-icons/core/regular/check-circle.svg?raw';
  import checkMark from '@phosphor-icons/core/bold/check-bold.svg?raw';
  import alert from '@phosphor-icons/core/regular/warning-circle.svg?raw';
  import copy from '@phosphor-icons/core/regular/copy.svg?raw';
  import arrowUp from '@phosphor-icons/core/bold/arrow-up-bold.svg?raw';
  import arrowDown from '@phosphor-icons/core/bold/arrow-down-bold.svg?raw';
  import settings from '@phosphor-icons/core/regular/gear-six.svg?raw';
  import sun from '@phosphor-icons/core/regular/sun.svg?raw';
  import moon from '@phosphor-icons/core/regular/moon.svg?raw';
  import monitor from '@phosphor-icons/core/regular/monitor.svg?raw';
  import shield from '@phosphor-icons/core/regular/shield-check.svg?raw';
  import info from '@phosphor-icons/core/regular/info.svg?raw';
  import terminal from '@phosphor-icons/core/regular/terminal-window.svg?raw';
  import timer from '@phosphor-icons/core/regular/timer.svg?raw';
  import link from '@phosphor-icons/core/regular/link-simple.svg?raw';
  import lock from '@phosphor-icons/core/fill/lock-simple-fill.svg?raw';
  import lockOpen from '@phosphor-icons/core/regular/lock-simple-open.svg?raw';
  import tunnel from '@phosphor-icons/core/regular/arrows-left-right.svg?raw';
  import funnel from '@phosphor-icons/core/regular/funnel-simple.svg?raw';
  import diagram from '@phosphor-icons/core/regular/tree-structure.svg?raw';
  import zoomIn from '@phosphor-icons/core/regular/magnifying-glass-plus.svg?raw';
  import zoomOut from '@phosphor-icons/core/regular/magnifying-glass-minus.svg?raw';
  import fit from '@phosphor-icons/core/regular/corners-out.svg?raw';
  import follow from '@phosphor-icons/core/regular/arrow-right.svg?raw';
  import json from '@phosphor-icons/core/regular/brackets-curly.svg?raw';
  import restore from '@phosphor-icons/core/regular/arrow-counter-clockwise.svg?raw';
  import sortDefault from '@phosphor-icons/core/regular/arrows-down-up.svg?raw';
  import fitWidth from '@phosphor-icons/core/regular/arrows-out-line-horizontal.svg?raw';
  import fn from '@phosphor-icons/core/regular/function.svg?raw';
  import procedure from '@phosphor-icons/core/regular/gear-fine.svg?raw';
  import sequence from '@phosphor-icons/core/regular/list-numbers.svg?raw';
  import enumType from '@phosphor-icons/core/regular/list-bullets.svg?raw';
  import domain from '@phosphor-icons/core/regular/tag-simple.svg?raw';
  import trigger from '@phosphor-icons/core/regular/lightning.svg?raw';
  import extension from '@phosphor-icons/core/regular/puzzle-piece.svg?raw';
  import event from '@phosphor-icons/core/regular/clock.svg?raw';
  import ddl from '@phosphor-icons/core/regular/brackets-angle.svg?raw';

  const ICONS = {
    plus, x, search, table, view, refresh, play, stop,
    'chevron-left': chevronLeft, 'chevron-right': chevronRight, 'chevron-down': chevronDown, 'chevrons-left': chevronsLeft, 'chevrons-right': chevronsRight,
    database, unplug, trash, pencil, code, key, columns, rows, folder, check, alert, copy,
    arrowUp, arrowDown, settings, sun, moon, monitor, shield, info, terminal, timer, link, lock, lockOpen, tunnel, funnel,
    download, update: arrowCircleUp, 'check-mark': checkMark, diagram, zoomIn, zoomOut, fit, follow, json, restore, sortDefault, fitWidth,
    function: fn, procedure, sequence, enum: enumType, domain, trigger, extension, event, ddl,
  } as const;

  export type IconName = keyof typeof ICONS;
</script>

<script lang="ts">
  let { name, size = 14 }: { name: IconName; size?: number } = $props();
</script>

<span class="icon" style:width="{size}px" style:height="{size}px" aria-hidden="true">{@html ICONS[name]}</span>

<style>
  .icon { flex: none; display: inline-flex; }
  .icon :global(svg) { width: 100%; height: 100%; display: block; }
</style>
