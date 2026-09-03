import { describe, it, expect } from "vitest";
import {
  buildStructuredConstraintsFromForm,
  collapseEmptyStructuredConstraints,
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

  it("treats all-star patterns as wildcards", () => {
    expect(
      constraintsObjectHasNonWildcard({ title: { $pattern: "**" } }),
    ).toBe(false);
    expect(
      constraintsObjectHasNonWildcard({ title: { $pattern: "*" } }),
    ).toBe(false);
    expect(
      constraintsObjectHasNonWildcard({ title: { $pattern: "a*" } }),
    ).toBe(true);
    const wildScenario = emptyScenario();
    wildScenario.paramRows.title = [
      { ...emptyConstraintRow(), mode: "pattern", value: "**" },
    ];
    expect(
      formStateHasNonWildcardConstraint({ scenarios: [wildScenario] }),
    ).toBe(false);
    const mixed = emptyScenario();
    mixed.paramRows.repo = [
      { ...emptyConstraintRow(), value: "my-repo", mode: "fixed" },
    ];
    mixed.paramRows.title = [
      { ...emptyConstraintRow(), mode: "pattern", value: "**" },
    ];
    expect(formStateHasNonWildcardConstraint({ scenarios: [mixed] })).toBe(
      true,
    );
  });

  it("treats empty form state as unrestricted", () => {
    expect(
      formStateHasNonWildcardConstraint({ scenarios: [emptyScenario()] }),
    ).toBe(false);
  });

  it("collapses empty v2 documents to {}", () => {
    expect(
      collapseEmptyStructuredConstraints(
        buildStructuredConstraintsFromForm({ scenarios: [emptyScenario()] }),
      ),
    ).toEqual({});
    expect(
      collapseEmptyStructuredConstraints({ repo: "*" }),
    ).toEqual({ repo: "*" });
  });

  it("round-trips comparison operators", () => {
    const constraints = {
      $version: 2,
      match: "any",
      groups: [
        {
          match: "all",
          conditions: [{ field: "limit", op: "lte", value: 20 }],
        },
      ],
    };
    const form = constraintsToFormState(constraints);
    expect(form.scenarios[0]?.paramRows.limit?.[0]?.operator).toBe("lte");
    expect(form.scenarios[0]?.paramRows.limit?.[0]?.value).toBe("20");
    const rebuilt = buildStructuredConstraintsFromForm(form);
    expect(rebuilt).toEqual(constraints);
  });
});
