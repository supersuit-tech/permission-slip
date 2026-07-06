import { describe, it, expect } from "vitest";
import {
  buildStructuredConstraintsFromForm,
  constraintsObjectHasNonWildcard,
  constraintsToFormState,
  formStateHasNonWildcardConstraint,
  parseStructuredConstraintsForDisplay,
  emptyConstraintRow,
  emptyScenario,
} from "@/lib/structuredConstraints";

describe("structuredConstraints", () => {
  it("round-trips flat map through form state", () => {
    const flat = { repo: "supersuit-tech/ci-actions", title: "*" };
    const form = constraintsToFormState(flat);
    expect(form.scenarios).toHaveLength(1);
    expect(form.scenarios[0]?.paramRows.repo).toHaveLength(1);
    const built = buildStructuredConstraintsFromForm(form);
    expect(built.$version).toBe(2);
    expect(built.groups).toBeDefined();
  });

  it("builds negation and any_of conditions", () => {
    const scenario = emptyScenario();
    scenario.paramRows.channel = [
      { ...emptyConstraintRow("matches"), value: "#engineering", mode: "fixed" },
      { ...emptyConstraintRow("matches"), value: "#releases", mode: "fixed" },
      {
        ...emptyConstraintRow("does_not_match"),
        value: "#executive-only",
        mode: "fixed",
      },
    ];
    const built = buildStructuredConstraintsFromForm({ scenarios: [scenario] });
    const conditions = (
      built.groups as Array<{ conditions: Array<{ op: string }> }>
    )[0]?.conditions;
    expect(conditions?.some((c) => c.op === "any_of")).toBe(true);
    expect(
      conditions?.some(
        (c) => c.op === "none_of" || c.op === "does_not_match",
      ),
    ).toBe(true);
  });

  it("detects non-wildcard constraints in form state", () => {
    const scenario = emptyScenario();
    scenario.paramRows.repo = [
      { ...emptyConstraintRow(), value: "my-repo", mode: "fixed" },
    ];
    expect(
      formStateHasNonWildcardConstraint({ scenarios: [scenario] }),
    ).toBe(true);
  });

  it("displays negated constraints", () => {
    const constraints = {
      $version: 2,
      match: "any",
      groups: [
        {
          match: "all",
          conditions: [
            {
              field: "channel",
              op: "none_of",
              values: ["#secret"],
            },
          ],
        },
      ],
    };
    const lines = parseStructuredConstraintsForDisplay(constraints);
    expect(lines[0]?.negated).toBe(true);
  });

  it("detects non-wildcard in flat and v2 objects", () => {
    expect(
      constraintsObjectHasNonWildcard({ repo: "foo", title: "*" }),
    ).toBe(true);
    expect(
      constraintsObjectHasNonWildcard({
        $version: 2,
        match: "any",
        groups: [
          {
            match: "all",
            conditions: [
              { field: "channel", op: "none_of", values: ["#secret"] },
            ],
          },
        ],
      }),
    ).toBe(true);
    expect(constraintsObjectHasNonWildcard({ title: "*" })).toBe(false);
  });
});
