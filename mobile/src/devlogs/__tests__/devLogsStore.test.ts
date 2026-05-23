import {
  __getDevLogsForTests,
  __resetDevLogsForTests,
  appendDevLog,
  clearDevLogs,
  nextDevLogId,
  type DevLogEntry,
} from "../devLogsStore";

function makeEntry(overrides: Partial<DevLogEntry> = {}): DevLogEntry {
  return {
    id: nextDevLogId(),
    method: "GET",
    url: "https://example.com/api/v1/x",
    status: 200,
    durationMs: 12,
    startedAt: Date.now(),
    body: "{}",
    isError: false,
    ...overrides,
  };
}

describe("devLogsStore", () => {
  beforeEach(() => {
    __resetDevLogsForTests();
  });

  it("evicts oldest entries when exceeding the 50-entry cap", () => {
    for (let i = 0; i < 80; i += 1) {
      appendDevLog(makeEntry({ url: `https://x/api/v1/r-${i}` }));
    }
    const snapshot = __getDevLogsForTests();
    expect(snapshot).toHaveLength(50);
    // Oldest 30 should be gone; the newest must include r-79.
    expect(snapshot[snapshot.length - 1]?.url).toBe(
      "https://x/api/v1/r-79",
    );
    expect(snapshot[0]?.url).toBe("https://x/api/v1/r-30");
  });

  it("truncates oversized bodies with a marker", () => {
    appendDevLog(makeEntry({ body: "x".repeat(10_000) }));
    const [entry] = __getDevLogsForTests();
    expect(entry).toBeDefined();
    expect(entry!.body.length).toBeLessThan(5_000);
    expect(entry!.body.endsWith("…(truncated)")).toBe(true);
  });

  it("clearDevLogs empties the buffer", () => {
    appendDevLog(makeEntry());
    appendDevLog(makeEntry());
    expect(__getDevLogsForTests()).toHaveLength(2);
    clearDevLogs();
    expect(__getDevLogsForTests()).toHaveLength(0);
  });

  it("nextDevLogId returns unique values", () => {
    const ids = new Set([
      nextDevLogId(),
      nextDevLogId(),
      nextDevLogId(),
    ]);
    expect(ids.size).toBe(3);
  });
});
