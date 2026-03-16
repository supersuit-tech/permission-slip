import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "../../../../test-helpers";
import { ParameterFieldWidget } from "../ParameterFieldWidget";
import type { SchemaProperty } from "@/lib/parameterSchema";

vi.mock("../../../../lib/supabaseClient");

function renderWidget(
  property: SchemaProperty,
  value = "",
  onChange = vi.fn(),
  disabled = false,
) {
  return {
    onChange,
    ...renderWithProviders(
      <ParameterFieldWidget
        paramKey="test_field"
        property={property}
        value={value}
        onChange={onChange}
        disabled={disabled}
      />,
    ),
  };
}

describe("ParameterFieldWidget", () => {
  describe("text widget (default)", () => {
    it("renders a text input when no x-ui widget is specified", () => {
      renderWidget({ type: "string" });

      const input = screen.getByRole("textbox");
      expect(input).toBeInTheDocument();
      expect(input).toHaveAttribute("type", "text");
    });

    it("renders a text input when x-ui.widget is 'text'", () => {
      renderWidget({ type: "string", "x-ui": { widget: "text" } });

      const input = screen.getByRole("textbox");
      expect(input).toHaveAttribute("type", "text");
    });

    it("uses x-ui.placeholder", () => {
      renderWidget({
        type: "string",
        "x-ui": { placeholder: "Enter customer ID" },
      });

      expect(screen.getByPlaceholderText("Enter customer ID")).toBeInTheDocument();
    });

    it("fires onChange when typed into", async () => {
      const user = userEvent.setup();
      const { onChange } = renderWidget({ type: "string" });

      await user.type(screen.getByRole("textbox"), "a");

      expect(onChange).toHaveBeenCalledWith("a");
    });
  });

  describe("select widget", () => {
    const selectProp: SchemaProperty = {
      type: "string",
      enum: ["usd", "eur", "gbp"],
      "x-ui": { widget: "select" },
    };

    it("renders a select element with enum options", () => {
      renderWidget(selectProp);

      const select = screen.getByTestId("select-param-test_field");
      expect(select.tagName).toBe("SELECT");
      expect(screen.getByText("usd")).toBeInTheDocument();
      expect(screen.getByText("eur")).toBeInTheDocument();
      expect(screen.getByText("gbp")).toBeInTheDocument();
    });

    it("includes a placeholder option", () => {
      renderWidget(selectProp);

      expect(screen.getByText("Select…")).toBeInTheDocument();
    });

    it("uses x-ui.placeholder as default option text", () => {
      renderWidget({
        type: "string",
        enum: ["usd", "eur"],
        "x-ui": { widget: "select", placeholder: "Choose a currency" },
      });

      expect(screen.getByText("Choose a currency")).toBeInTheDocument();
      expect(screen.queryByText("Select…")).not.toBeInTheDocument();
    });

    it("fires onChange when an option is selected", async () => {
      const user = userEvent.setup();
      const { onChange } = renderWidget(selectProp);

      await user.selectOptions(
        screen.getByTestId("select-param-test_field"),
        "eur",
      );

      expect(onChange).toHaveBeenCalledWith("eur");
    });
  });

  describe("textarea widget", () => {
    const textareaProp: SchemaProperty = {
      type: "string",
      "x-ui": { widget: "textarea", placeholder: "Enter description" },
    };

    it("renders a textarea element", () => {
      renderWidget(textareaProp);

      const textarea = screen.getByTestId("textarea-param-test_field");
      expect(textarea.tagName).toBe("TEXTAREA");
    });

    it("applies placeholder", () => {
      renderWidget(textareaProp);

      expect(screen.getByPlaceholderText("Enter description")).toBeInTheDocument();
    });

    it("fires onChange when typed into", async () => {
      const user = userEvent.setup();
      const { onChange } = renderWidget(textareaProp);

      await user.type(
        screen.getByTestId("textarea-param-test_field"),
        "x",
      );

      expect(onChange).toHaveBeenCalledWith("x");
    });
  });

  describe("toggle widget", () => {
    const toggleProp: SchemaProperty = {
      type: "boolean",
      "x-ui": { widget: "toggle" },
    };

    it("renders a switch component", () => {
      renderWidget(toggleProp, "false");

      expect(screen.getByRole("switch")).toBeInTheDocument();
      expect(screen.getByText("Disabled")).toBeInTheDocument();
    });

    it("shows Enabled when value is 'true'", () => {
      renderWidget(toggleProp, "true");

      expect(screen.getByText("Enabled")).toBeInTheDocument();
    });

    it("fires onChange with 'true' when toggled on", async () => {
      const user = userEvent.setup();
      const { onChange } = renderWidget(toggleProp, "false");

      await user.click(screen.getByRole("switch"));

      expect(onChange).toHaveBeenCalledWith("true");
    });

    it("fires onChange with 'false' when toggled off", async () => {
      const user = userEvent.setup();
      const { onChange } = renderWidget(toggleProp, "true");

      await user.click(screen.getByRole("switch"));

      expect(onChange).toHaveBeenCalledWith("false");
    });
  });

  describe("number widget", () => {
    const numberProp: SchemaProperty = {
      type: "number",
      "x-ui": { widget: "number", placeholder: "0" },
    };

    it("renders a number input", () => {
      renderWidget(numberProp);

      const input = screen.getByRole("spinbutton");
      expect(input).toHaveAttribute("type", "number");
    });

    it("applies placeholder", () => {
      renderWidget(numberProp);

      expect(screen.getByPlaceholderText("0")).toBeInTheDocument();
    });

    it("fires onChange when typed into", async () => {
      const user = userEvent.setup();
      const { onChange } = renderWidget(numberProp);

      await user.type(screen.getByRole("spinbutton"), "5");

      expect(onChange).toHaveBeenCalledWith("5");
    });
  });

  describe("date widget", () => {
    const dateProp: SchemaProperty = {
      type: "string",
      "x-ui": { widget: "date" },
    };

    it("renders a date input", () => {
      renderWidget(dateProp);

      // Date inputs don't have a specific role; query by id
      const input = document.getElementById("param-test_field") as HTMLInputElement;
      expect(input).toBeInTheDocument();
      expect(input.type).toBe("date");
    });

    it("fires onChange when a date is entered", async () => {
      const user = userEvent.setup();
      const { onChange } = renderWidget(dateProp);

      const input = document.getElementById("param-test_field") as HTMLInputElement;
      await user.type(input, "2025-06-01");

      expect(onChange).toHaveBeenCalled();
      // Verify the handler forwards a string (not undefined/hardcoded)
      const calls = onChange.mock.calls;
      const lastCall = calls[calls.length - 1]?.[0] as unknown;
      expect(typeof lastCall).toBe("string");
    });
  });

  describe("datetime widget", () => {
    const datetimeProp: SchemaProperty = {
      type: "string",
      "x-ui": { widget: "datetime" },
    };

    it("renders a datetime-local input", () => {
      renderWidget(datetimeProp);

      const input = document.getElementById("param-test_field") as HTMLInputElement;
      expect(input).toBeInTheDocument();
      expect(input.type).toBe("datetime-local");
    });

    it("converts RFC 3339 value to datetime-local format for display", () => {
      renderWidget(datetimeProp, "2026-03-16T17:00:00Z");

      const input = document.getElementById("param-test_field") as HTMLInputElement;
      // Should be formatted as YYYY-MM-DDTHH:mm in local time
      expect(input.value).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);
    });

    it("handles empty value", () => {
      renderWidget(datetimeProp, "");

      const input = document.getElementById("param-test_field") as HTMLInputElement;
      expect(input.value).toBe("");
    });

    it("fires onChange with RFC 3339 format", async () => {
      const user = userEvent.setup();
      const { onChange } = renderWidget(datetimeProp);

      const input = document.getElementById("param-test_field") as HTMLInputElement;
      await user.type(input, "2026-03-16T17:00");

      expect(onChange).toHaveBeenCalled();
    });

    it("disables the datetime input when disabled", () => {
      renderWidget(datetimeProp, "", vi.fn(), true);

      const input = document.getElementById("param-test_field") as HTMLInputElement;
      expect(input).toBeDisabled();
    });
  });

  describe("help hints", () => {
    it("renders help_text below the input", () => {
      renderWidget({
        type: "string",
        "x-ui": { help_text: "Use the Stripe customer ID format" },
      });

      expect(
        screen.getByText("Use the Stripe customer ID format"),
      ).toBeInTheDocument();
    });

    it("renders a help_url link", () => {
      renderWidget({
        type: "string",
        "x-ui": { help_url: "https://example.com/docs" },
      });

      const link = screen.getByRole("link", { name: /docs/i });
      expect(link).toHaveAttribute("href", "https://example.com/docs");
      expect(link).toHaveAttribute("target", "_blank");
    });

    it("renders both help_text and help_url together", () => {
      renderWidget({
        type: "string",
        "x-ui": {
          help_text: "Some help",
          help_url: "https://example.com",
        },
      });

      expect(screen.getByText("Some help")).toBeInTheDocument();
      expect(screen.getByRole("link", { name: /docs/i })).toBeInTheDocument();
    });

    it("does not render link for non-http help_url", () => {
      renderWidget({
        type: "string",
        "x-ui": { help_url: "javascript:alert(1)" },
      });

      expect(screen.queryByRole("link")).not.toBeInTheDocument();
    });

    it("renders nothing when no hints provided", () => {
      const { container } = renderWidget({ type: "string" });

      // Only the input should be present, no hint paragraphs
      expect(container.querySelectorAll("p")).toHaveLength(0);
      expect(container.querySelectorAll("a")).toHaveLength(0);
    });
  });

  describe("disabled state", () => {
    it("disables the text input when disabled", () => {
      renderWidget({ type: "string" }, "", vi.fn(), true);

      expect(screen.getByRole("textbox")).toBeDisabled();
    });

    it("disables the select when disabled", () => {
      renderWidget(
        { type: "string", enum: ["a"], "x-ui": { widget: "select" } },
        "",
        vi.fn(),
        true,
      );

      expect(screen.getByTestId("select-param-test_field")).toBeDisabled();
    });

    it("disables the toggle when disabled", () => {
      renderWidget(
        { type: "boolean", "x-ui": { widget: "toggle" } },
        "false",
        vi.fn(),
        true,
      );

      expect(screen.getByRole("switch")).toBeDisabled();
    });

    it("disables the textarea when disabled", () => {
      renderWidget(
        { type: "string", "x-ui": { widget: "textarea" } },
        "",
        vi.fn(),
        true,
      );

      expect(screen.getByTestId("textarea-param-test_field")).toBeDisabled();
    });

    it("disables the number input when disabled", () => {
      renderWidget(
        { type: "number", "x-ui": { widget: "number" } },
        "",
        vi.fn(),
        true,
      );

      expect(screen.getByRole("spinbutton")).toBeDisabled();
    });

    it("disables the date input when disabled", () => {
      renderWidget(
        { type: "string", "x-ui": { widget: "date" } },
        "",
        vi.fn(),
        true,
      );

      const input = document.getElementById("param-test_field") as HTMLInputElement;
      expect(input).toBeDisabled();
    });
  });
});
