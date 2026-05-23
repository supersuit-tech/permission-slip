/**
 * Parses a duration string like "30d", "12h", "90m" into seconds.
 * Supported units: d (days), h (hours), m (minutes), s (seconds).
 */
export function parseDurationToSeconds(input: string): number {
  const trimmed = input.trim();
  if (!trimmed) {
    throw new Error("duration must not be empty");
  }
  const match = /^(\d+)([dhms])$/i.exec(trimmed);
  if (!match) {
    throw new Error(`invalid duration "${input}"; use forms like 30d, 12h, 90m, 3600s`);
  }
  const value = Number.parseInt(match[1]!, 10);
  const unit = match[2]!.toLowerCase();
  const multipliers: Record<string, number> = {
    d: 86400,
    h: 3600,
    m: 60,
    s: 1,
  };
  const mult = multipliers[unit];
  if (!mult) {
    throw new Error(`unsupported duration unit in "${input}"`);
  }
  return value * mult;
}
