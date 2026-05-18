import { renderHook } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { useResourceLimit } from "../useResourceLimit";

describe("useResourceLimit", () => {
  it("returns unlimited limits with the caller-supplied count", () => {
    const { result } = renderHook(() =>
      useResourceLimit("max_agents", 3),
    );
    expect(result.current.hasData).toBe(true);
    expect(result.current.max).toBeNull();
    expect(result.current.current).toBe(3);
    expect(result.current.atLimit).toBe(false);
  });
});
