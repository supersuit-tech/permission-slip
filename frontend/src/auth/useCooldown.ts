import { useState, useEffect, useRef, useCallback } from "react";

interface UseCooldownResult {
  secondsLeft: number;
  isActive: boolean;
  start: (seconds?: number) => void;
}

const DEFAULT_SECONDS = 60;

/** Tracks a countdown timer (e.g. for rate-limit cooldowns). Call start() to
 *  begin a countdown; secondsLeft counts down to 0, then isActive becomes false. */
export function useCooldown(): UseCooldownResult {
  const [secondsLeft, setSecondsLeft] = useState(0);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (intervalRef.current !== null) clearInterval(intervalRef.current);
    };
  }, []);

  const start = useCallback((seconds = DEFAULT_SECONDS) => {
    if (seconds <= 0) return;
    if (intervalRef.current !== null) clearInterval(intervalRef.current);
    setSecondsLeft(seconds);
    intervalRef.current = setInterval(() => {
      setSecondsLeft((prev) => {
        if (prev <= 1) {
          if (intervalRef.current !== null) clearInterval(intervalRef.current);
          intervalRef.current = null;
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  }, []);

  return { secondsLeft, isActive: secondsLeft > 0, start };
}
