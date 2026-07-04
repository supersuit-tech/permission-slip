import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useConnectorDisplayName } from "../useConnectorDisplayName";
import { createQueryClient, mockSession, waitFor } from "../../__test-utils__";

const mockGet = jest.fn();

jest.mock("../../api/client", () => ({
  __esModule: true,
  default: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}));

jest.mock("../../auth/AuthContext", () => ({
  useAuth: () => ({
    session: mockSession(),
  }),
}));

interface HookCapture {
  connectorDisplayName: string;
  isLoading: boolean;
}

function createHookCapture(actionType: string) {
  const capture: HookCapture = {
    connectorDisplayName: "",
    isLoading: true,
  };

  function Consumer() {
    const result = useConnectorDisplayName(actionType);
    capture.connectorDisplayName = result.connectorDisplayName;
    capture.isLoading = result.isLoading;
    return null;
  }

  return { capture, Consumer };
}

function renderWithProviders(Consumer: React.ComponentType, qc: QueryClient) {
  return create(
    createElement(
      QueryClientProvider,
      { client: qc },
      createElement(Consumer),
    ),
  );
}

let currentRenderer: ReactTestRenderer | null = null;
let currentQueryClient: QueryClient | null = null;

describe("useConnectorDisplayName", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(async () => {
    if (currentQueryClient) {
      currentQueryClient.cancelQueries();
      currentQueryClient.clear();
      currentQueryClient = null;
    }
    if (currentRenderer) {
      await act(async () => {
        currentRenderer!.unmount();
      });
      currentRenderer = null;
    }
  });

  it("returns connector manifest name when available", async () => {
    mockGet.mockResolvedValue({
      data: { id: "protonmail", name: "Proton Mail", actions: [] },
      error: undefined,
    });

    const { capture, Consumer } = createHookCapture("protonmail.read_email");
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    await waitFor(() => capture.connectorDisplayName === "Proton Mail");
    expect(capture.isLoading).toBe(false);
  });

  it("falls back to humanized prefix when connector detail is unavailable", async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { message: "not found" },
    });

    const { capture, Consumer } = createHookCapture("protonmail.read_email");
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    await waitFor(() => capture.isLoading === false);
    expect(capture.connectorDisplayName).toBe("Protonmail");
  });
});
