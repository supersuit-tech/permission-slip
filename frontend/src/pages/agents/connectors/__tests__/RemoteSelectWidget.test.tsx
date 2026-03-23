import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "../../../../test-helpers";
import { RemoteSelectWidget } from "../RemoteSelectWidget";

vi.mock("../../../../lib/supabaseClient");

const mockUseCalendars = vi.fn();

vi.mock("@/hooks/useAgentConnectorCalendars", () => ({
  useAgentConnectorCalendars: () => mockUseCalendars(),
}));

const baseUi = {
  remote_select_options_path:
    "/v1/agents/{agent_id}/connectors/{connector_id}/calendars",
  remote_select_id_key: "id",
  remote_select_label_key: "summary",
  help_text: "Connect a credential to select a calendar.",
} as const;

describe("RemoteSelectWidget", () => {
  it("shows disabled helper when no credential", () => {
    mockUseCalendars.mockReturnValue({
      calendars: [],
      isLoading: false,
      isFetching: false,
      error: null,
      hasCredential: false,
      isCredentialBindingPending: false,
      refetch: vi.fn(),
    });

    renderWithProviders(
      <RemoteSelectWidget
        inputId="param-calendar_id"
        value=""
        onChange={vi.fn()}
        agentId={1}
        connectorId="google"
        ui={baseUi}
      />,
    );

    expect(screen.getByTestId("remote-select-disabled-param-calendar_id")).toBeDisabled();
    expect(
      screen.getByText("Connect a credential to select a calendar."),
    ).toBeInTheDocument();
  });

  it("renders options when credential exists", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    mockUseCalendars.mockReturnValue({
      calendars: [
        { id: "primary", summary: "Primary", primary: true },
        { id: "work@x", summary: "Work" },
      ],
      isLoading: false,
      isFetching: false,
      error: null,
      hasCredential: true,
      isCredentialBindingPending: false,
      refetch: vi.fn(),
    });

    renderWithProviders(
      <RemoteSelectWidget
        inputId="param-calendar_id"
        value=""
        onChange={onChange}
        agentId={1}
        connectorId="google"
        ui={baseUi}
      />,
    );

    const select = screen.getByTestId("remote-select-param-calendar_id");
    await user.selectOptions(select, "work@x");
    expect(onChange).toHaveBeenCalledWith("work@x");
    expect(screen.getByRole("option", { name: /Primary \(primary\)/ })).toBeInTheDocument();
  });

  it("switches to manual entry on button click", async () => {
    const user = userEvent.setup();
    mockUseCalendars.mockReturnValue({
      calendars: [{ id: "a", summary: "A" }],
      isLoading: false,
      isFetching: false,
      error: null,
      hasCredential: true,
      isCredentialBindingPending: false,
      refetch: vi.fn(),
    });

    renderWithProviders(
      <RemoteSelectWidget
        inputId="param-calendar_id"
        value=""
        onChange={vi.fn()}
        agentId={1}
        connectorId="google"
        ui={baseUi}
      />,
    );

    await user.click(screen.getByRole("button", { name: /enter manually/i }));
    expect(
      screen.getByTestId("remote-select-manual-param-calendar_id"),
    ).toBeInTheDocument();
  });

  it("shows checking credentials while binding is loading", () => {
    mockUseCalendars.mockReturnValue({
      calendars: [],
      isLoading: false,
      isFetching: false,
      error: null,
      hasCredential: false,
      isCredentialBindingPending: true,
      refetch: vi.fn(),
    });

    renderWithProviders(
      <RemoteSelectWidget
        inputId="param-calendar_id"
        value=""
        onChange={vi.fn()}
        agentId={1}
        connectorId="google"
        ui={baseUi}
      />,
    );

    expect(
      screen.getByTestId("remote-select-binding-pending-param-calendar_id"),
    ).toBeDisabled();
    expect(screen.getByText("Checking credentials…")).toBeInTheDocument();
  });
});
