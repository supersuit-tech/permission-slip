import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { ConstraintsSummary } from "../ConstraintsSummary";

describe("ConstraintsSummary", () => {
  it("renders 'No constraints' when constraints is null", () => {
    render(<ConstraintsSummary constraints={null} />);
    expect(screen.getByText("No constraints")).toBeInTheDocument();
  });

  it("renders 'No constraints' when constraints is empty object", () => {
    render(<ConstraintsSummary constraints={{}} />);
    expect(screen.getByText("No constraints")).toBeInTheDocument();
  });

  it("renders fixed constraint with Lock icon and value", () => {
    render(<ConstraintsSummary constraints={{ repo: "my-repo" }} />);
    expect(screen.getByText("repo")).toBeInTheDocument();
    expect(screen.getByText("my-repo")).toBeInTheDocument();
  });

  it("renders wildcard constraint showing 'any'", () => {
    render(<ConstraintsSummary constraints={{ subject: "*" }} />);
    expect(screen.getByText("subject")).toBeInTheDocument();
    expect(screen.getByText("any")).toBeInTheDocument();
  });

  it("renders pattern constraint with $pattern wrapper", () => {
    render(
      <ConstraintsSummary
        constraints={{ to: { $pattern: "*@mycompany.com" } }}
      />,
    );
    expect(screen.getByText("to")).toBeInTheDocument();
    expect(screen.getByText("*@mycompany.com")).toBeInTheDocument();
  });

  it("truncates long values", () => {
    const longValue = "a".repeat(30);
    render(<ConstraintsSummary constraints={{ field: longValue }} />);
    // Should be truncated (23 chars + ellipsis)
    expect(screen.queryByText(longValue)).not.toBeInTheDocument();
  });

  it("shows '+N more' when more than 2 constraints", () => {
    render(
      <ConstraintsSummary
        constraints={{
          repo: "my-repo",
          branch: "main",
          owner: "me",
          label: "bug",
        }}
      />,
    );
    expect(screen.getByText("repo")).toBeInTheDocument();
    expect(screen.getByText("branch")).toBeInTheDocument();
    expect(screen.getByText("+2 more")).toBeInTheDocument();
    // Hidden constraints
    expect(screen.queryByText("owner")).not.toBeInTheDocument();
  });

  it("expands to show all constraints on click", () => {
    render(
      <ConstraintsSummary
        constraints={{
          repo: "my-repo",
          branch: "main",
          owner: "me",
        }}
      />,
    );
    fireEvent.click(screen.getByText("+1 more"));
    expect(screen.getByText("owner")).toBeInTheDocument();
    expect(screen.getByText("show less")).toBeInTheDocument();
  });

  it("collapses back on 'show less' click", () => {
    render(
      <ConstraintsSummary
        constraints={{
          repo: "my-repo",
          branch: "main",
          owner: "me",
        }}
      />,
    );
    fireEvent.click(screen.getByText("+1 more"));
    fireEvent.click(screen.getByText("show less"));
    expect(screen.queryByText("owner")).not.toBeInTheDocument();
    expect(screen.getByText("+1 more")).toBeInTheDocument();
  });
});
