/**
 * Tests for the shell quoting utility.
 */

import { shellQuote } from "../src/util/shell.js";

describe("shellQuote", () => {
  it("wraps a simple string in single quotes", () => {
    expect(shellQuote("hello")).toBe("'hello'");
  });

  it("handles a URL with no special characters", () => {
    expect(shellQuote("https://app.permissionslip.dev")).toBe(
      "'https://app.permissionslip.dev'",
    );
  });

  it("escapes embedded single quotes", () => {
    expect(shellQuote("it's a test")).toBe("'it'\\''s a test'");
  });

  it("handles multiple embedded single quotes", () => {
    expect(shellQuote("a'b'c")).toBe("'a'\\''b'\\''c'");
  });

  it("handles shell metacharacters safely", () => {
    const value = 'echo $HOME; rm -rf /';
    expect(shellQuote(value)).toBe("'echo $HOME; rm -rf /'");
  });

  it("handles an empty string", () => {
    expect(shellQuote("")).toBe("''");
  });
});
