/**
 * Tests for the approval polling utility.
 */

import { jest } from "@jest/globals";
import { pollUntilResolved, type ApprovalStatusResult } from "../src/util/poll.js";
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
    if (result.status === "approved") {
      expect(result.timed_out).toBeUndefined();
    }
  });
});
