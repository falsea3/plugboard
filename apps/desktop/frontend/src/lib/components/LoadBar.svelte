<script lang="ts">
  let { label = 'Loading' }: { label?: string } = $props();
</script>

<div class="load-bar" role="progressbar" aria-label={label}><span></span></div>

<style>
  .load-bar {
    position: absolute;
    z-index: 5;
    top: 0;
    left: 0;
    right: 0;
    height: 2px;
    overflow: hidden;
    pointer-events: none;
    opacity: 0;
    animation: show 0.15s 0.2s forwards;
  }
  .load-bar span {
    position: absolute;
    inset: 0 auto 0 0;
    width: 30%;
    border-radius: 2px;
    background: var(--accent);
    animation: slide 1.1s ease-in-out infinite;
  }
  @keyframes show { to { opacity: 1; } }
  @keyframes slide {
    from { transform: translateX(-100%); }
    to { transform: translateX(334%); }
  }
  @media (prefers-reduced-motion: reduce) { .load-bar span { animation-duration: 2.4s; } }
</style>
