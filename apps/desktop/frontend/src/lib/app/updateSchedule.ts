export const CHECK_EVERY = 15 * 60_000;
export const CHECK_ON_FOCUS_AFTER = 5 * 60_000;
const FIRST_CHECK = 3000;

type Clock = { now: () => number; online: () => boolean };

const browserClock: Clock = {
  now: () => Date.now(),
  online: () => navigator.onLine !== false,
};

export function scheduleUpdateChecks(
  check: () => Promise<unknown>,
  target: Pick<Window, "addEventListener" | "removeEventListener"> = window,
  clock = browserClock,
): () => void {
  let last = 0;
  let running = false;
  const run = async (minGap: number) => {
    if (running || !clock.online() || (last && clock.now() - last < minGap))
      return;
    running = true;
    last = clock.now();
    try {
      await check();
    } finally {
      running = false;
    }
  };
  const first = setTimeout(() => run(0), FIRST_CHECK);
  const timer = setInterval(() => run(CHECK_EVERY / 2), CHECK_EVERY);
  const onFocus = () => void run(CHECK_ON_FOCUS_AFTER);
  target.addEventListener("focus", onFocus);
  return () => {
    clearTimeout(first);
    clearInterval(timer);
    target.removeEventListener("focus", onFocus);
  };
}
