import { useRef, useCallback, type KeyboardEvent } from "react";

interface Option<T extends string> {
  label: string;
  value: T;
}

interface SegmentedControlProps<T extends string> {
  options: Option<T>[];
  value: T;
  onChange: (v: T) => void;
  ariaLabel: string;
}

export function SegmentedControl<T extends string>({
  options,
  value,
  onChange,
  ariaLabel,
}: SegmentedControlProps<T>) {
  const groupRef = useRef<HTMLDivElement>(null);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLButtonElement>) => {
      const idx = options.findIndex((o) => o.value === value);
      let next = -1;

      if (e.key === "ArrowRight" || e.key === "ArrowDown") {
        next = (idx + 1) % options.length;
      } else if (e.key === "ArrowLeft" || e.key === "ArrowUp") {
        next = (idx - 1 + options.length) % options.length;
      } else if (e.key === "Home") {
        next = 0;
      } else if (e.key === "End") {
        next = options.length - 1;
      }

      if (next >= 0) {
        e.preventDefault();
        const opt = options[next];
        if (opt) {
          onChange(opt.value);
          const buttons = groupRef.current?.querySelectorAll<HTMLButtonElement>(
            '[role="radio"]',
          );
          buttons?.[next]?.focus();
        }
      }
    },
    [options, value, onChange],
  );

  return (
    <div
      ref={groupRef}
      className="inline-flex gap-0.5 rounded-lg bg-muted p-1"
      role="radiogroup"
      aria-label={ariaLabel}
    >
      {options.map((opt) => (
        <button
          key={opt.value}
          type="button"
          role="radio"
          aria-checked={value === opt.value}
          tabIndex={value === opt.value ? 0 : -1}
          onClick={() => onChange(opt.value)}
          onKeyDown={handleKeyDown}
          className={
            value === opt.value
              ? "rounded-md bg-background px-3 py-1.5 text-sm font-medium text-foreground shadow-sm focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:outline-none"
              : "rounded-md px-3 py-1.5 text-sm font-medium text-muted-foreground hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:outline-none"
          }
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}
