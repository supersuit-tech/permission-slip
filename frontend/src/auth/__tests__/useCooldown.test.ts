import { renderHook, act } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useCooldown } from "../useCooldown";

let clearIntervalSpy: ReturnType<typeof vi.spyOn> | null = null;

describe("useCooldown", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    clearIntervalSpy?.mockRestore();
    clearIntervalSpy = null;
  });

  it("starts inactive with secondsLeft at 0", () => {
    const { result } = renderHook(() => useCooldown());
    expect(result.current.secondsLeft).toBe(0);
    expect(result.current.isActive).toBe(false);
  });

  it("becomes active and counts down after start()", () => {
    const { result } = renderHook(() => useCooldown());

    act(() => {
      result.current.start(3);
    });

    expect(result.current.secondsLeft).toBe(3);
    expect(result.current.isActive).toBe(true);

    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(result.current.secondsLeft).toBe(2);

    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(result.current.secondsLeft).toBe(1);
  });

  it("reaches 0 and becomes inactive after full countdown", () => {
    const { result } = renderHook(() => useCooldown());

    act(() => {
      result.current.start(2);
    });

    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(result.current.secondsLeft).toBe(0);
    expect(result.current.isActive).toBe(false);
  });

  it("resets correctly when start() is called while already running", () => {
    const { result } = renderHook(() => useCooldown());

    act(() => {
      result.current.start(10);
    });

    act(() => {
      vi.advanceTimersByTime(3000);
    });
    expect(result.current.secondsLeft).toBe(7);

    // Restart with a fresh 5-second countdown
    act(() => {
      result.current.start(5);
    });

    expect(result.current.secondsLeft).toBe(5);

    act(() => {
      vi.advanceTimersByTime(5000);
    });

    expect(result.current.secondsLeft).toBe(0);
    expect(result.current.isActive).toBe(false);
  });

  it("does nothing when start(0) is called", () => {
    const { result } = renderHook(() => useCooldown());

    act(() => {
      result.current.start(0);
    });

    expect(result.current.secondsLeft).toBe(0);
    expect(result.current.isActive).toBe(false);
  });

  it("cleans up interval on unmount mid-countdown", () => {
    clearIntervalSpy = vi.spyOn(globalThis, "clearInterval");
    const { result, unmount } = renderHook(() => useCooldown());

    act(() => {
      result.current.start(10);
    });

    unmount();

    expect(clearIntervalSpy).toHaveBeenCalled();
  });
});
