import { vi } from "vitest";

export const mockGet = vi.fn();
export const mockPost = vi.fn();
export const mockPatch = vi.fn();
export const mockPut = vi.fn();
export const mockDelete = vi.fn();

/**
 * Reset all client method mocks. Call in beforeEach.
 */
export function resetClientMocks() {
  mockGet.mockReset();
  mockPost.mockReset();
  mockPatch.mockReset();
  mockPut.mockReset();
  mockDelete.mockReset();
}

export default {
  GET: (...args: unknown[]) => mockGet(...args),
  POST: (...args: unknown[]) => mockPost(...args),
  PATCH: (...args: unknown[]) => mockPatch(...args),
  PUT: (...args: unknown[]) => mockPut(...args),
  DELETE: (...args: unknown[]) => mockDelete(...args),
};
