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
  opts?: { siblingDatetimeValue?: string },
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
        siblingDatetimeValue={opts?.siblingDatetimeValue}
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

    it("renders dynamic calendar options when options_from is connector_calendars", () => {
      const calendarProp: SchemaProperty = {
        type: "string",
        "x-ui": {
          widget: "select",
          options_from: "connector_calendars",
          placeholder: "Pick one",
        },
      };

      renderWithProviders(
        <ParameterFieldWidget
          paramKey="calendar_id"
          property={calendarProp}
          value=""
          onChange={vi.fn()}
          dynamicSelectOptions={[
            { value: "primary", label: "Primary (primary)" },
            { value: "other", label: "Other" },
          ]}
          dynamicSelectLoading={false}
        />,
      );

      const select = screen.getByTestId("select-param-calendar_id");
      expect(select).toBeInTheDocument();
      expect(screen.getByText("Primary (primary)")).toBeInTheDocument();
      expect(screen.getByText("Other")).toBeInTheDocument();
    });

    it("shows loading placeholder for dynamic calendar select", () => {
      const calendarProp: SchemaProperty = {
        type: "string",
        "x-ui": { widget: "select", options_from: "connector_calendars" },
      };

      renderWithProviders(
        <ParameterFieldWidget
          paramKey="calendar_id"
          property={calendarProp}
          value=""
          onChange={vi.fn()}
          dynamicSelectOptions={[]}
          dynamicSelectLoading
        />,
      );

      expect(screen.getByText("Loading calendars…")).toBeInTheDocument();
    });

    it("falls back to text input when dynamic calendar loading fails", () => {
      const calendarProp: SchemaProperty = {
        type: "string",
        "x-ui": { widget: "select", options_from: "connector_calendars" },
      };

      renderWithProviders(
        <ParameterFieldWidget
          paramKey="calendar_id"
          property={calendarProp}
          value=""
          onChange={vi.fn()}
          dynamicSelectLoading={false}
          dynamicSelectError="Failed to load calendars"
        />,
      );

      // Should render a text input instead of a select
      const input = screen.getByTestId("select-param-calendar_id");
      expect(input.tagName).toBe("INPUT");
      expect(input).toHaveAttribute("type", "text");
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

    it("sets min from sibling when datetime_range_role is upper", () => {
      const prop: SchemaProperty = {
        type: "string",
        "x-ui": {
          widget: "datetime",
          datetime_range_role: "upper",
          datetime_range_pair: "time_min",
        },
      };

      renderWidget(prop, "", vi.fn(), false, {
        siblingDatetimeValue: "2026-03-16T12:00:00-05:00",
      });

      const input = document.getElementById("param-test_field") as HTMLInputElement;
      expect(input.min).toMatch(/^2026-03-16T\d{2}:\d{2}$/);
    });

    it("sets max from sibling when datetime_range_role is lower", () => {
      const prop: SchemaProperty = {
        type: "string",
        "x-ui": {
          widget: "datetime",
          datetime_range_role: "lower",
          datetime_range_pair: "time_max",
        },
      };

      renderWidget(prop, "", vi.fn(), false, {
        siblingDatetimeValue: "2026-03-20T15:00:00Z",
      });

      const input = document.getElementById("param-test_field") as HTMLInputElement;
      expect(input.max).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);
    });

    it("does not set min/max when sibling is a wildcard pattern", () => {
      const prop: SchemaProperty = {
        type: "string",
        "x-ui": {
          widget: "datetime",
          datetime_range_role: "upper",
        },
      };

      renderWidget(prop, "", vi.fn(), false, {
        siblingDatetimeValue: "2026-*",
      });

      const input = document.getElementById("param-test_field") as HTMLInputElement;
      expect(input.min).toBe("");
      expect(input.max).toBe("");
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

  describe("list widget", () => {
    const listProp: SchemaProperty = {
      type: "array",
      items: { type: "string" },
      "x-ui": { widget: "list" },
    };

    it("renders an add item button when value is empty", () => {
      renderWidget(listProp, "");

      expect(screen.getByRole("button", { name: /add item/i })).toBeInTheDocument();
    });

    it("renders items from a JSON array value", () => {
      renderWidget(listProp, '["tag1","tag2"]');

      const inputs = screen.getAllByRole("textbox");
      expect(inputs).toHaveLength(2);
      expect(inputs[0]).toHaveValue("tag1");
      expect(inputs[1]).toHaveValue("tag2");
    });

    it("adds a new empty item when Add item is clicked", async () => {
      const user = userEvent.setup();
      const { onChange } = renderWidget(listProp, "");

      await user.click(screen.getByRole("button", { name: /add item/i }));

      // Should serialize with the empty item so a row appears for the user to type into
      expect(onChange).toHaveBeenCalledWith('[""]');
    });

    it("removes an item when the remove button is clicked", async () => {
      const user = userEvent.setup();
      const { onChange } = renderWidget(listProp, '["a","b"]');

      const removeButtons = screen.getAllByRole("button", { name: /remove item/i });
      await user.click(removeButtons[0]!);

      expect(onChange).toHaveBeenCalledWith('["b"]');
    });

    it("fires onChange with updated JSON when an item is edited", async () => {
      const user = userEvent.setup();
      const { onChange } = renderWidget(listProp, '["a"]');

      const input = screen.getByRole("textbox");
      await user.type(input, "b");

      // Last call should serialize the appended value
      const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1]?.[0];
      expect(lastCall).toBe('["ab"]');
    });

    it("disables inputs and buttons when disabled", () => {
      renderWidget(listProp, '["a"]', vi.fn(), true);

      expect(screen.getByRole("textbox")).toBeDisabled();
      expect(screen.getByRole("button", { name: /remove item/i })).toBeDisabled();
      expect(screen.getByRole("button", { name: /add item/i })).toBeDisabled();
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
