import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  CHECK_EVERY,
  CHECK_ON_FOCUS_AFTER,
  scheduleUpdateChecks,
} from "./updateSchedule";

describe("scheduleUpdateChecks", () => {
  let target: EventTarget;
  let online = true;
  const clock = { now: () => Date.now(), online: () => online };

  beforeEach(() => {
    vi.useFakeTimers();
    target = new EventTarget();
    online = true;
  });
  afterEach(() => vi.useRealTimers());

  it("checks soon after launch, then on an interval", async () => {
    const check = vi.fn(async () => {});
    const stop = scheduleUpdateChecks(check, target as Window, clock);
    await vi.advanceTimersByTimeAsync(3000);
    expect(check).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(CHECK_EVERY);
    expect(check).toHaveBeenCalledTimes(2);
    stop();
    await vi.advanceTimersByTimeAsync(CHECK_EVERY * 3);
    expect(check).toHaveBeenCalledTimes(2);
  });

  it("checks on focus only when the last check is old enough", async () => {
    const check = vi.fn(async () => {});
    scheduleUpdateChecks(check, target as Window, clock);
    await vi.advanceTimersByTimeAsync(3000);
    target.dispatchEvent(new Event("focus"));
    await vi.advanceTimersByTimeAsync(0);
    expect(check).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(CHECK_ON_FOCUS_AFTER);
    target.dispatchEvent(new Event("focus"));
    await vi.advanceTimersByTimeAsync(0);
    expect(check).toHaveBeenCalledTimes(2);
  });

  it("skips while offline and never runs two checks at once", async () => {
    let finish = () => {};
    const check = vi.fn(() => new Promise<void>((r) => (finish = r)));
    online = false;
    scheduleUpdateChecks(check, target as Window, clock);
    await vi.advanceTimersByTimeAsync(3000);
    expect(check).not.toHaveBeenCalled();
    online = true;
    await vi.advanceTimersByTimeAsync(CHECK_EVERY);
    expect(check).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(CHECK_EVERY);
    expect(check).toHaveBeenCalledTimes(1);
    finish();
    await vi.advanceTimersByTimeAsync(CHECK_EVERY);
    expect(check).toHaveBeenCalledTimes(2);
  });
});
