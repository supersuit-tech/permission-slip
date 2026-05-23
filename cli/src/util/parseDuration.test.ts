import { parseDurationToSeconds } from "./parseDuration.js";

describe("parseDurationToSeconds", () => {
  it("parses days", () => {
    expect(parseDurationToSeconds("30d")).toBe(30 * 86400);
  });

  it("parses hours", () => {
    expect(parseDurationToSeconds("12h")).toBe(12 * 3600);
  });

  it("parses minutes and seconds", () => {
    expect(parseDurationToSeconds("90m")).toBe(5400);
    expect(parseDurationToSeconds("45s")).toBe(45);
  });

  it("rejects invalid input", () => {
    expect(() => parseDurationToSeconds("")).toThrow();
    expect(() => parseDurationToSeconds("30x")).toThrow();
    expect(() => parseDurationToSeconds("abc")).toThrow();
  });
});
