import { constraintsAreUnrestricted } from "../unrestrictedConstraints";

describe("constraintsAreUnrestricted", () => {
  it("treats empty objects and all-star patterns as unrestricted", () => {
    expect(constraintsAreUnrestricted({})).toBe(true);
    expect(constraintsAreUnrestricted({ title: "*" })).toBe(true);
    expect(constraintsAreUnrestricted({ title: { $pattern: "**" } })).toBe(true);
    expect(constraintsAreUnrestricted({ title: { $pattern: "a*" } })).toBe(false);
    expect(constraintsAreUnrestricted({ repo: "foo" })).toBe(false);
    expect(
      constraintsAreUnrestricted({
        $meta: { from: "*" },
        message_id: "*",
      }),
    ).toBe(true);
    expect(
      constraintsAreUnrestricted({
        $meta: { from: "a@example.com" },
        message_id: "*",
      }),
    ).toBe(false);
  });
});
