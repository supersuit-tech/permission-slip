/**
 * Tests for the approval polling utility.
 */

import { jest } from "@jest/globals";
import { pollUntilResolved, parseTimeout, type ApprovalStatusResult } from "../src/util/poll.js";
import { PermissionSlipApiError } from "../src/api/client.js";
import type { ApiClient } from "../src/api/client.js";

function makeResult(overrides: Partial<ApprovalStatusResult> = {}): ApprovalStatusResult {
  return {
    approval_id: "appr_test123",
    status: "pending",
    expires_at: "2026-03-14T00:00:00Z",
    created_at: "2026-03-13T00:00:00Z",
    ...overrides,
  };
}

function makeMockClient(results: ApprovalStatusResult[]): { client: ApiClient; mock: jest.Mock } {
  let callIndex = 0;
  const mock = jest.fn<() => Promise<ApprovalStatusResult>>(async () => {
    const result = results[Math.min(callIndex, results.length - 1)]!;
    callIndex++;
    return result;
  });
  return {
    client: { approvalStatus: mock } as unknown as ApiClient,
    mock,
  };
}

describe("pollUntilResolved", () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it("returns immediately when status is already approved", async () => {
    const approved = makeResult({ status: "approved", execution_status: "success", execution_result: { url: "https://example.com" } });
    const { client, mock } = makeMockClient([approved]);

    const result = await pollUntilResolved({
      approvalId: "appr_test123",
      client,
      timeoutSeconds: 10,
    });

    expect(result.status).toBe("approved");
    expect(result.execution_status).toBe("success");
    expect(result.execution_result).toEqual({ url: "https://example.com" });
    expect(result.timed_out).toBeUndefined();
    expect(mock).toHaveBeenCalledTimes(1);
  });

  it("returns immediately when status is denied", async () => {
    const denied = makeResult({ status: "denied" });
    const { client } = makeMockClient([denied]);

    const result = await pollUntilResolved({
      approvalId: "appr_test123",
      client,
      timeoutSeconds: 10,
    });

    expect(result.status).toBe("denied");
    expect(result.timed_out).toBeUndefined();
  });

  it("returns immediately when status is cancelled", async () => {
    const cancelled = makeResult({ status: "cancelled" });
    const { client } = makeMockClient([cancelled]);

    const result = await pollUntilResolved({
      approvalId: "appr_test123",
      client,
      timeoutSeconds: 10,
    });

    expect(result.status).toBe("cancelled");
    expect(result.timed_out).toBeUndefined();
  });

  it("returns immediately when status is expired", async () => {
    const expired = makeResult({ status: "expired" });
    const { client } = makeMockClient([expired]);

    const result = await pollUntilResolved({
      approvalId: "appr_test123",
      client,
      timeoutSeconds: 10,
    });

    expect(result.status).toBe("expired");
    expect(result.timed_out).toBeUndefined();
  });

  it("polls multiple times then resolves when status changes", async () => {
    const pending = makeResult({ status: "pending" });
    const approved = makeResult({ status: "approved", execution_status: "success" });
    const { client, mock } = makeMockClient([pending, pending, approved]);

    const promise = pollUntilResolved({
      approvalId: "appr_test123",
      client,
      timeoutSeconds: 30,
    });

    // Advance through the sleep intervals to trigger each poll.
    // Poll 1 (immediate) → pending → sleep 2s
    await jest.advanceTimersByTimeAsync(2000);
    // Poll 2 → pending → sleep 3s
    await jest.advanceTimersByTimeAsync(3000);
    // Poll 3 → approved → returns

    const result = await promise;

    expect(result.status).toBe("approved");
    expect(result.timed_out).toBeUndefined();
    expect(mock.mock.calls.length).toBeGreaterThanOrEqual(3);
  });

  it("times out and returns timed_out flag", async () => {
    const pending = makeResult({ status: "pending" });
    const { client } = makeMockClient([pending]);

    const promise = pollUntilResolved({
      approvalId: "appr_test123",
      client,
      timeoutSeconds: 3,
    });

    // Advance past the full timeout + extra to ensure it completes.
    await jest.advanceTimersByTimeAsync(5000);

    const result = await promise;

    expect(result.status).toBe("pending");
    expect(result.timed_out).toBe(true);
  });

  it("returns resolved status on final check even after timeout", async () => {
    let callCount = 0;
    const mock = jest.fn<() => Promise<ApprovalStatusResult>>(async () => {
      callCount++;
      // Return approved on the final check (3rd call)
      if (callCount >= 3) {
        return makeResult({ status: "approved", execution_status: "success" });
      }
      return makeResult({ status: "pending" });
    });
    const client = { approvalStatus: mock } as unknown as ApiClient;

    const promise = pollUntilResolved({
      approvalId: "appr_test123",
      client,
      timeoutSeconds: 3,
    });

    // Advance enough for the timeout to elapse so the final check fires.
    await jest.advanceTimersByTimeAsync(5000);

    const result = await promise;

    // The final check after timeout returns approved, so timed_out should not be set.
    expect(result.status).toBe("approved");
    expect(result.timed_out).toBeUndefined();
  });

  it("retries on transient errors and resolves", async () => {
    let callCount = 0;
    const mock = jest.fn<() => Promise<ApprovalStatusResult>>(async () => {
      callCount++;
      if (callCount === 1) {
        // First call: transient 503
        throw new PermissionSlipApiError(503, {
          code: "service_unavailable",
          message: "Service Unavailable",
          retryable: true,
        });
      }
      // Second call: approved
      return makeResult({ status: "approved", execution_status: "success" });
    });
    const client = { approvalStatus: mock } as unknown as ApiClient;

    const promise = pollUntilResolved({
      approvalId: "appr_test123",
      client,
      timeoutSeconds: 10,
    });

    // Advance past first sleep interval after transient error.
    await jest.advanceTimersByTimeAsync(2000);

    const result = await promise;

    expect(result.status).toBe("approved");
    expect(result.timed_out).toBeUndefined();
    expect(mock.mock.calls.length).toBe(2);
  });

  it("retries on native network errors (TypeError) and resolves", async () => {
    let callCount = 0;
    const mock = jest.fn<() => Promise<ApprovalStatusResult>>(async () => {
      callCount++;
      if (callCount === 1) {
        // First call: network-level failure (e.g., ECONNREFUSED)
        throw new TypeError("fetch failed");
      }
      return makeResult({ status: "approved", execution_status: "success" });
    });
    const client = { approvalStatus: mock } as unknown as ApiClient;

    const promise = pollUntilResolved({
      approvalId: "appr_test123",
      client,
      timeoutSeconds: 10,
    });

    await jest.advanceTimersByTimeAsync(2000);

    const result = await promise;

    expect(result.status).toBe("approved");
    expect(result.timed_out).toBeUndefined();
    expect(mock.mock.calls.length).toBe(2);
  });

  it("returns timed_out with minimal info when final check fails transiently", async () => {
    const mock = jest.fn<() => Promise<ApprovalStatusResult>>(async () => {
      // Always throw transient error — even on the final post-timeout check.
      throw new TypeError("fetch failed");
    });
    const client = { approvalStatus: mock } as unknown as ApiClient;

    const promise = pollUntilResolved({
      approvalId: "appr_test123",
      client,
      timeoutSeconds: 1,
    });

    await jest.advanceTimersByTimeAsync(3000);

    const result = await promise;

    expect(result.status).toBe("pending");
    expect(result.timed_out).toBe(true);
    expect(result.approval_id).toBe("appr_test123");
    // Should have minimal info (empty strings) since the API never responded.
    expect(result.expires_at).toBe("");
    expect(result.created_at).toBe("");
  });

  it("calls onPoll with correct arguments while pending", async () => {
    const pending = makeResult({ status: "pending" });
    const approved = makeResult({ status: "approved", execution_status: "success" });
    const { client } = makeMockClient([pending, approved]);
    const pollCalls: Array<{ status: string; elapsed: number; timeout: number }> = [];

    const promise = pollUntilResolved({
      approvalId: "appr_test123",
      client,
      timeoutSeconds: 30,
      onPoll: (info) => pollCalls.push(info),
    });

    // Poll 1 → pending → onPoll called → sleep 2s
    await jest.advanceTimersByTimeAsync(2000);
    // Poll 2 → approved → returns

    await promise;

    expect(pollCalls).toHaveLength(1);
    expect(pollCalls[0]!.status).toBe("pending");
    expect(pollCalls[0]!.timeout).toBe(30);
    expect(typeof pollCalls[0]!.elapsed).toBe("number");
  });

  it("uses fixed interval when fixedIntervalSeconds is set", async () => {
    let callCount = 0;
    const mock = jest.fn<() => Promise<ApprovalStatusResult>>(async () => {
      callCount++;
      if (callCount >= 4) {
        return makeResult({ status: "approved", execution_status: "success" });
      }
      return makeResult({ status: "pending" });
    });
    const client = { approvalStatus: mock } as unknown as ApiClient;

    const promise = pollUntilResolved({
      approvalId: "appr_test123",
      client,
      timeoutSeconds: 60,
      fixedIntervalSeconds: 5,
    });

    // Poll 1 (immediate) → pending → sleep 5s
    await jest.advanceTimersByTimeAsync(5000);
    // Poll 2 → pending → sleep 5s (fixed, no backoff)
    await jest.advanceTimersByTimeAsync(5000);
    // Poll 3 → pending → sleep 5s (still fixed)
    await jest.advanceTimersByTimeAsync(5000);
    // Poll 4 → approved

    const result = await promise;

    expect(result.status).toBe("approved");
    expect(result.timed_out).toBeUndefined();
    expect(mock.mock.calls.length).toBe(4);
  });

  it("fixed interval polls respect timeout", async () => {
    const pending = makeResult({ status: "pending" });
    const { client } = makeMockClient([pending]);

    const promise = pollUntilResolved({
      approvalId: "appr_test123",
      client,
      timeoutSeconds: 7,
      fixedIntervalSeconds: 5,
    });

    // Advance past timeout
    await jest.advanceTimersByTimeAsync(10000);

    const result = await promise;

    expect(result.status).toBe("pending");
    expect(result.timed_out).toBe(true);
  });

  it("propagates non-transient errors immediately", async () => {
    const mock = jest.fn<() => Promise<ApprovalStatusResult>>(async () => {
      throw new PermissionSlipApiError(401, {
        code: "unauthorized",
        message: "Unauthorized",
        retryable: false,
      });
    });
    const client = { approvalStatus: mock } as unknown as ApiClient;

    await expect(
      pollUntilResolved({
        approvalId: "appr_test123",
        client,
        timeoutSeconds: 10,
      }),
    ).rejects.toThrow("Unauthorized");

    expect(mock.mock.calls.length).toBe(1);
  });
});

describe("parseTimeout", () => {
  const noop = () => {};

  it("returns default for undefined", () => {
    expect(parseTimeout(undefined, noop)).toBe(120);
  });

  it("returns default for non-numeric", () => {
    expect(parseTimeout("abc", noop)).toBe(120);
  });

  it("returns default for negative value", () => {
    expect(parseTimeout("-5", noop)).toBe(120);
  });

  it("returns default for zero", () => {
    expect(parseTimeout("0", noop)).toBe(120);
  });

  it("clamps large values to maximum", () => {
    expect(parseTimeout("100000", noop)).toBe(86400);
  });

  it("parses valid timeout", () => {
    expect(parseTimeout("30", noop)).toBe(30);
  });

  it("warns on non-numeric value", () => {
    const warnings: string[] = [];
    parseTimeout("abc", (msg) => warnings.push(msg));
    expect(warnings).toHaveLength(1);
    expect(warnings[0]).toContain("invalid --timeout");
    expect(warnings[0]).toContain("abc");
  });

  it("warns on zero value", () => {
    const warnings: string[] = [];
    parseTimeout("0", (msg) => warnings.push(msg));
    expect(warnings).toHaveLength(1);
    expect(warnings[0]).toContain("invalid --timeout");
  });

  it("warns on negative value", () => {
    const warnings: string[] = [];
    parseTimeout("-5", (msg) => warnings.push(msg));
    expect(warnings).toHaveLength(1);
    expect(warnings[0]).toContain("invalid --timeout");
  });

  it("does not warn on valid value", () => {
    const warnings: string[] = [];
    parseTimeout("30", (msg) => warnings.push(msg));
    expect(warnings).toHaveLength(0);
  });

  it("uses custom flag name in warning", () => {
    const warnings: string[] = [];
    parseTimeout("bad", (msg) => warnings.push(msg), { flagName: "--poll-timeout" });
    expect(warnings).toHaveLength(1);
    expect(warnings[0]).toContain("invalid --poll-timeout");
    expect(warnings[0]).not.toContain("invalid --timeout ");
  });

  it("uses custom default timeout", () => {
    expect(parseTimeout(undefined, noop, { defaultTimeout: 600 })).toBe(600);
  });

  it("uses custom default on invalid value", () => {
    expect(parseTimeout("bad", noop, { defaultTimeout: 600 })).toBe(600);
  });
});
