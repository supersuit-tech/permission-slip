import { vi } from "vitest";

export { Toaster } from "sonner";

export const toast = {
  error: vi.fn(),
  success: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  message: vi.fn(),
};
