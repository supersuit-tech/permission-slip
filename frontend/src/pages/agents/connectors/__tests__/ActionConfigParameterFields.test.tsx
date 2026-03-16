import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "../../../../test-helpers";
import { ActionConfigParameterFields } from "../ActionConfigParameterFields";
import type { ParametersSchema } from "@/lib/parameterSchema";
import type { ParamMode } from "../ActionConfigFormFields";

vi.mock("../../../../lib/supabaseClient");

function renderFields(
  schema: ParametersSchema | null,
  overrides?: {
    values?: Record<string, string>;
    modes?: Record<string, ParamMode>;
    onValueChange?: (key: string, value: string) => void;
    onModeChange?: (key: string, mode: ParamMode) => void;
    disabled?: boolean;
  },
) {
  const onValueChange = overrides?.onValueChange ?? vi.fn<(key: string, value: string) => void>();
  const onModeChange = overrides?.onModeChange ?? vi.fn<(key: string, mode: ParamMode) => void>();
  return {
    onValueChange,
    onModeChange,
    ...renderWithProviders(
      <ActionConfigParameterFields
        parametersSchema={schema}
        values={overrides?.values ?? {}}
        onValueChange={onValueChange}
        modes={overrides?.modes ?? {}}
        onModeChange={onModeChange}
        disabled={overrides?.disabled}
      />,
    ),
  };
}

const basicSchema: ParametersSchema = {
  type: "object",
  required: ["name"],
  properties: {
    name: { type: "string", description: "The name" },
    email: { type: "string" },
  },
};

describe("ActionConfigParameterFields", () => {
  describe("backwards compatibility (no x-ui)", () => {
    it("renders fields when no x-ui hints are present", () => {
      renderFields(basicSchema);

      expect(screen.getByLabelText("name")).toBeInTheDocument();
      expect(screen.getByLabelText("email")).toBeInTheDocument();
    });

    it("shows 'no configurable parameters' when properties are absent", () => {
      renderFields({ type: "object" });

      expect(
        screen.getByText("This action has no configurable parameters."),
      ).toBeInTheDocument();
    });

    it("shows 'no configurable parameters' for null schema", () => {
      renderFields(null);

      expect(
        screen.getByText("This action has no configurable parameters."),
      ).toBeInTheDocument();
    });

    it("shows required badge for required fields", () => {
      renderFields(basicSchema);

      expect(screen.getByText("required")).toBeInTheDocument();
    });

    it("shows type annotation", () => {
      renderFields(basicSchema);

      // Both fields are type: "string", so there should be two type annotations
      expect(screen.getAllByText("(string)")).toHaveLength(2);
    });

    it("shows description text", () => {
      renderFields(basicSchema);

      expect(screen.getByText("The name")).toBeInTheDocument();
    });
  });

  describe("x-ui.order", () => {
    it("orders fields according to x-ui.order", () => {
      const schema: ParametersSchema = {
        type: "object",
        properties: {
          alpha: { type: "string" },
          beta: { type: "string" },
          gamma: { type: "string" },
        },
        "x-ui": { order: ["gamma", "alpha", "beta"] },
      };

      renderFields(schema);

      const labels = screen.getAllByText(/^(alpha|beta|gamma)$/);
      expect(labels.map((l) => l.textContent)).toEqual(["gamma", "alpha", "beta"]);
    });

    it("appends fields not in order at the end", () => {
      const schema: ParametersSchema = {
        type: "object",
        properties: {
          alpha: { type: "string" },
          beta: { type: "string" },
          gamma: { type: "string" },
        },
        "x-ui": { order: ["gamma"] },
      };

      renderFields(schema);

      const labels = screen.getAllByText(/^(alpha|beta|gamma)$/);
      expect(labels.map((l) => l.textContent)).toEqual(["gamma", "alpha", "beta"]);
    });
  });

  describe("x-ui.label", () => {
    it("uses x-ui.label as display label", () => {
      const schema: ParametersSchema = {
        type: "object",
        properties: {
          customer_id: {
            type: "string",
            "x-ui": { label: "Customer" },
          },
        },
      };

      renderFields(schema);

      expect(screen.getByText("Customer")).toBeInTheDocument();
      expect(screen.queryByText("customer_id")).not.toBeInTheDocument();
    });

    it("falls back to property key when no x-ui.label", () => {
      renderFields(basicSchema);

      expect(screen.getByText("name")).toBeInTheDocument();
    });
  });

  describe("x-ui.placeholder", () => {
    it("uses x-ui.placeholder in fixed mode", () => {
      const schema: ParametersSchema = {
        type: "object",
        properties: {
          customer_id: {
            type: "string",
            "x-ui": { placeholder: "cus_ABC123" },
          },
        },
      };

      renderFields(schema);

      expect(screen.getByPlaceholderText("cus_ABC123")).toBeInTheDocument();
    });

    it("overrides x-ui.placeholder in wildcard mode", () => {
      const schema: ParametersSchema = {
        type: "object",
        properties: {
          customer_id: {
            type: "string",
            "x-ui": { placeholder: "cus_ABC123" },
          },
        },
      };

      renderFields(schema, {
        values: { customer_id: "*" },
        modes: { customer_id: "wildcard" },
      });

      expect(
        screen.getByPlaceholderText("Agent can use any value"),
      ).toBeInTheDocument();
      expect(screen.queryByPlaceholderText("cus_ABC123")).not.toBeInTheDocument();
    });

    it("uses x-ui.placeholder in pattern mode (no override)", () => {
      const schema: ParametersSchema = {
        type: "object",
        properties: {
          customer_id: {
            type: "string",
            "x-ui": { placeholder: "cus_ABC123" },
          },
        },
      };

      renderFields(schema, {
        modes: { customer_id: "pattern" },
      });

      // Pattern mode no longer overrides the placeholder — the field uses x-ui.placeholder
      expect(screen.getByPlaceholderText("cus_ABC123")).toBeInTheDocument();
    });
  });

  describe("x-ui.groups (collapsible sections)", () => {
    const groupedSchema: ParametersSchema = {
      type: "object",
      properties: {
        customer_id: {
          type: "string",
          "x-ui": { label: "Customer", group: "billing" },
        },
        currency: {
          type: "string",
          enum: ["usd", "eur"],
          "x-ui": { widget: "select", group: "billing" },
        },
        auto_advance: {
          type: "boolean",
          "x-ui": { widget: "toggle", group: "options" },
        },
        note: { type: "string" },
      },
      "x-ui": {
        groups: [
          { id: "billing", label: "Billing" },
          { id: "options", label: "Options", collapsed: true, description: "Advanced settings" },
        ],
      },
    };

    it("renders ungrouped fields outside group sections", () => {
      renderFields(groupedSchema);

      // "note" is ungrouped, should be visible directly
      expect(screen.getByLabelText("note")).toBeInTheDocument();
    });

    it("renders group headers", () => {
      renderFields(groupedSchema);

      expect(screen.getByRole("button", { name: /Billing/ })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: /Options/ })).toBeInTheDocument();
    });

    it("shows group description", () => {
      renderFields(groupedSchema);

      expect(screen.getByText(/Advanced settings/)).toBeInTheDocument();
    });

    it("shows grouped fields in expanded groups", () => {
      renderFields(groupedSchema);

      // Billing is not collapsed — its fields should be visible
      expect(screen.getByText("Customer")).toBeInTheDocument();
      expect(screen.getByText("currency")).toBeInTheDocument();
    });

    it("hides fields in collapsed groups until expanded", async () => {
      const user = userEvent.setup();
      renderFields(groupedSchema);

      // Options is collapsed — auto_advance should not be visible
      expect(screen.queryByText("auto_advance")).not.toBeInTheDocument();

      // Click to expand
      await user.click(screen.getByRole("button", { name: /Options/ }));

      // Now auto_advance should be visible
      expect(screen.getByText("auto_advance")).toBeInTheDocument();
    });

    it("hides group when all its fields are hidden by visible_when", () => {
      const schema: ParametersSchema = {
        type: "object",
        properties: {
          mode: { type: "string" },
          secret: {
            type: "string",
            "x-ui": {
              group: "advanced",
              visible_when: { field: "mode", equals: "advanced" },
            },
          },
        },
        "x-ui": {
          groups: [{ id: "advanced", label: "Advanced" }],
        },
      };

      renderFields(schema, { values: { mode: "simple" } });

      // The group header should not render when all children are hidden
      expect(screen.queryByText("Advanced")).not.toBeInTheDocument();
    });

    it("collapses an expanded group when clicked", async () => {
      const user = userEvent.setup();
      renderFields(groupedSchema);

      // Billing starts expanded
      expect(screen.getByText("Customer")).toBeInTheDocument();

      // Click to collapse
      await user.click(screen.getByRole("button", { name: /Billing/ }));

      // Customer field should now be hidden
      expect(screen.queryByText("Customer")).not.toBeInTheDocument();
    });
  });

  describe("visible_when", () => {
    const conditionalSchema: ParametersSchema = {
      type: "object",
      properties: {
        mode: { type: "string", enum: ["simple", "advanced"] },
        advanced_setting: {
          type: "string",
          "x-ui": {
            visible_when: { field: "mode", equals: "advanced" },
          },
        },
      },
    };

    it("hides field when condition is not met", () => {
      renderFields(conditionalSchema, {
        values: { mode: "simple" },
      });

      expect(screen.queryByLabelText("advanced_setting")).not.toBeInTheDocument();
    });

    it("shows field when condition is met", () => {
      renderFields(conditionalSchema, {
        values: { mode: "advanced" },
      });

      expect(screen.getByLabelText("advanced_setting")).toBeInTheDocument();
    });

    it("shows field when no visible_when is set", () => {
      renderFields(conditionalSchema, {
        values: { mode: "simple" },
      });

      expect(screen.getByLabelText("mode")).toBeInTheDocument();
    });

    it("matches boolean equals via string coercion", () => {
      const schema: ParametersSchema = {
        type: "object",
        properties: {
          enabled: { type: "boolean" },
          detail: {
            type: "string",
            "x-ui": {
              visible_when: { field: "enabled", equals: true },
            },
          },
        },
      };

      // Form values are strings — "true" should match boolean true
      renderFields(schema, { values: { enabled: "true" } });
      expect(screen.getByLabelText("detail")).toBeInTheDocument();
    });

    it("hides field when boolean equals does not match", () => {
      const schema: ParametersSchema = {
        type: "object",
        properties: {
          enabled: { type: "boolean" },
          detail: {
            type: "string",
            "x-ui": {
              visible_when: { field: "enabled", equals: true },
            },
          },
        },
      };

      renderFields(schema, { values: { enabled: "false" } });
      expect(screen.queryByLabelText("detail")).not.toBeInTheDocument();
    });

    it("matches number equals via coercion", () => {
      const schema: ParametersSchema = {
        type: "object",
        properties: {
          count: { type: "number" },
          bonus: {
            type: "string",
            "x-ui": {
              visible_when: { field: "count", equals: 5 },
            },
          },
        },
      };

      renderFields(schema, { values: { count: "5" } });
      expect(screen.getByLabelText("bonus")).toBeInTheDocument();
    });
  });

  describe("widget integration", () => {
    it("renders select widget for x-ui.widget='select'", () => {
      const schema: ParametersSchema = {
        type: "object",
        properties: {
          currency: {
            type: "string",
            enum: ["usd", "eur"],
            "x-ui": { widget: "select" },
          },
        },
      };

      renderFields(schema);

      expect(screen.getByTestId("select-param-currency")).toBeInTheDocument();
    });

    it("renders toggle widget for x-ui.widget='toggle'", () => {
      const schema: ParametersSchema = {
        type: "object",
        properties: {
          enabled: {
            type: "boolean",
            "x-ui": { widget: "toggle" },
          },
        },
      };

      renderFields(schema, { values: { enabled: "false" } });

      expect(screen.getByRole("switch")).toBeInTheDocument();
    });

    it("renders textarea widget for x-ui.widget='textarea'", () => {
      const schema: ParametersSchema = {
        type: "object",
        properties: {
          body: {
            type: "string",
            "x-ui": { widget: "textarea" },
          },
        },
      };

      renderFields(schema);

      expect(screen.getByTestId("textarea-param-body")).toBeInTheDocument();
    });
  });

  describe("constraint mode interaction", () => {
    it("defaults to fixed mode when no explicit mode is set, even with * value", () => {
      renderFields(basicSchema, {
        values: { name: "*" },
        modes: {},
      });

      // The input should NOT be disabled (fixed mode allows editing)
      const input = screen.getByLabelText("name");
      expect(input).not.toBeDisabled();
      // The value should show * as-is
      expect(input).toHaveValue("*");
    });

    it("passes value changes through to onValueChange", async () => {
      const user = userEvent.setup();
      const onValueChange = vi.fn<(key: string, value: string) => void>();
      renderFields(basicSchema, { onValueChange });

      await user.type(screen.getByLabelText("name"), "a");

      expect(onValueChange).toHaveBeenCalledWith("name", "a");
    });

    it("checking 'Any value' sets wildcard mode and value", async () => {
      const user = userEvent.setup();
      const onModeChange = vi.fn<(key: string, mode: ParamMode) => void>();
      const onValueChange = vi.fn<(key: string, value: string) => void>();
      renderFields(basicSchema, { onModeChange, onValueChange });

      const checkbox = screen.getAllByRole("checkbox")[0]!;
      await user.click(checkbox);

      expect(onModeChange).toHaveBeenCalledWith("name", "wildcard");
      expect(onValueChange).toHaveBeenCalledWith("name", "*");
    });

    it("unchecking 'Any value' sets fixed mode and clears value", async () => {
      const user = userEvent.setup();
      const onModeChange = vi.fn<(key: string, mode: ParamMode) => void>();
      const onValueChange = vi.fn<(key: string, value: string) => void>();
      renderFields(basicSchema, {
        onModeChange,
        onValueChange,
        values: { name: "*" },
        modes: { name: "wildcard" },
      });

      const checkbox = screen.getAllByRole("checkbox")[0]!;
      await user.click(checkbox);

      expect(onModeChange).toHaveBeenCalledWith("name", "fixed");
      expect(onValueChange).toHaveBeenCalledWith("name", "");
    });

    it("disables input when mode is wildcard", () => {
      renderFields(basicSchema, {
        values: { name: "*" },
        modes: { name: "wildcard" },
      });

      expect(screen.getByLabelText("name")).toBeDisabled();
    });

    it("shows wildcard hint when value contains *", () => {
      renderFields(basicSchema, {
        values: { name: "*@company.com" },
      });

      expect(screen.getByText("matches any text")).toBeInTheDocument();
    });

    it("does not show wildcard hint when mode is wildcard", () => {
      renderFields(basicSchema, {
        values: { name: "*" },
        modes: { name: "wildcard" },
      });

      expect(screen.queryByText("matches any text")).not.toBeInTheDocument();
    });
  });
});
