import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "../../../../test-helpers";
import { TemplatePicker } from "../TemplatePicker";
import type { ActionConfigTemplate } from "../../../../hooks/useActionConfigTemplates";

vi.mock("../../../../lib/supabaseClient");

const mockTemplates: ActionConfigTemplate[] = [
  {
    id: "tpl_1",
    connector_id: "github",
    action_type: "github.create_issue",
    name: "Create issues (all fields open)",
    description: "Agent can create issues in any repo with any title and body.",
    parameters: { repo: "*", title: "*", body: "*" },
    created_at: "2026-02-28T10:00:00Z",
  },
  {
    id: "tpl_2",
    connector_id: "github",
    action_type: "github.merge_pr",
    name: "Merge PRs",
    description: "Agent can merge any PR",
    parameters: { repo: "*", pull_number: "*" },
    created_at: "2026-02-28T10:00:00Z",
  },
  {
    id: "tpl_3",
    connector_id: "github",
    action_type: "github.create_issue",
    name: "Create issues in org",
    description: null,
    parameters: {
      repo: { $pattern: "myorg-*" },
      title: "*",
    },
    created_at: "2026-02-28T10:00:00Z",
  },
];

describe("TemplatePicker", () => {
  it("renders templates filtered by action type", () => {
    renderWithProviders(
      <TemplatePicker
        templates={mockTemplates}
        isLoading={false}
        actionType="github.create_issue"
        onSelect={vi.fn()}
      />,
    );

    expect(screen.getByText("Create issues (all fields open)")).toBeInTheDocument();
    expect(screen.getByText("Create issues in org")).toBeInTheDocument();
    // merge_pr template should NOT appear.
    expect(screen.queryByText("Merge PRs")).not.toBeInTheDocument();
  });

  it("renders nothing when no templates match the action type", () => {
    const { container } = renderWithProviders(
      <TemplatePicker
        templates={mockTemplates}
        isLoading={false}
        actionType="github.some_other_action"
        onSelect={vi.fn()}
      />,
    );

    expect(container.textContent).toBe("");
  });

  it("shows loading state", () => {
    renderWithProviders(
      <TemplatePicker
        templates={[]}
        isLoading={true}
        actionType="github.create_issue"
        onSelect={vi.fn()}
      />,
    );

    expect(screen.getByText("Loading templates...")).toBeInTheDocument();
  });

  it("calls onSelect when a template is clicked", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();

    renderWithProviders(
      <TemplatePicker
        templates={mockTemplates}
        isLoading={false}
        actionType="github.create_issue"
        onSelect={onSelect}
      />,
    );

    await user.click(screen.getByText("Create issues (all fields open)"));

    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(onSelect).toHaveBeenCalledWith(mockTemplates[0]);
  });

  it("shows parameter badges with wildcard styling", () => {
    renderWithProviders(
      <TemplatePicker
        templates={mockTemplates}
        isLoading={false}
        actionType="github.create_issue"
        onSelect={vi.fn()}
      />,
    );

    // Wildcard params should be displayed as "*"
    const wildcardBadges = screen.getAllByText("*");
    expect(wildcardBadges.length).toBeGreaterThan(0);
  });

  it("shows pattern parameter values", () => {
    renderWithProviders(
      <TemplatePicker
        templates={mockTemplates}
        isLoading={false}
        actionType="github.create_issue"
        onSelect={vi.fn()}
      />,
    );

    // Pattern values should show the glob string
    expect(screen.getByText("myorg-*")).toBeInTheDocument();
  });

  it("shows template descriptions", () => {
    renderWithProviders(
      <TemplatePicker
        templates={mockTemplates}
        isLoading={false}
        actionType="github.create_issue"
        onSelect={vi.fn()}
      />,
    );

    expect(
      screen.getByText("Agent can create issues in any repo with any title and body."),
    ).toBeInTheDocument();
  });

  it("shows applied indicator when a template is selected", () => {
    renderWithProviders(
      <TemplatePicker
        templates={mockTemplates}
        isLoading={false}
        actionType="github.create_issue"
        onSelect={vi.fn()}
        selectedTemplateId="tpl_1"
      />,
    );

    expect(screen.getByText("Applied")).toBeInTheDocument();
    // Only the selected template should show the applied badge
    const appliedBadges = screen.getAllByText("Applied");
    expect(appliedBadges).toHaveLength(1);
  });

  it("disables template buttons when disabled prop is true", () => {
    renderWithProviders(
      <TemplatePicker
        templates={mockTemplates}
        isLoading={false}
        actionType="github.create_issue"
        onSelect={vi.fn()}
        disabled={true}
      />,
    );

    const buttons = screen.getAllByRole("button");
    buttons.forEach((button) => {
      expect(button).toBeDisabled();
    });
  });
});
