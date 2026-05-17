import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../test-helpers";
import { makeTestAccessToken, setupAuthMocks } from "./fixtures";
import LoginPage from "../LoginPage";

describe("LoginPage", () => {
  beforeEach(() => {
    setupAuthMocks({ authenticated: false });
  });

  it("shows email and password fields in sign-in mode", async () => {
    renderWithProviders(<LoginPage />);
    await waitFor(() => {
      expect(screen.getByLabelText("Email")).toBeInTheDocument();
    });
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    const form = screen.getByRole("form");
    expect(within(form).getByRole("button", { name: "Sign in" })).toBeInTheDocument();
  });

  it("switches to create account mode", async () => {
    renderWithProviders(<LoginPage />);
    await waitFor(() => {
      expect(screen.getByText("Create account")).toBeInTheDocument();
    });
    await userEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(
      screen.getByText("Create an account with your email and password.")
    ).toBeInTheDocument();
    const createAccountButtons = screen.getAllByRole("button", {
      name: "Create account",
    });
    expect(createAccountButtons.length).toBeGreaterThanOrEqual(1);
  });

  it("submits sign-in and calls login endpoint", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/auth/login")) {
        return new Response(
          JSON.stringify({
            access_token: makeTestAccessToken("u1", "test@example.com"),
            refresh_token: "rt-new",
            expires_at: new Date(Date.now() + 900_000).toISOString(),
          }),
          { status: 200, headers: { "Content-Type": "application/json" } }
        );
      }
      return new Response("{}", { status: 404 });
    });
    globalThis.fetch = fetchMock as typeof fetch;

    renderWithProviders(<LoginPage />);
    await waitFor(() => {
      expect(screen.getByLabelText("Email")).toBeInTheDocument();
    });
    await userEvent.type(screen.getByLabelText("Email"), "test@example.com");
    await userEvent.type(screen.getByLabelText("Password"), "password12345");
    await userEvent.click(
      within(screen.getByRole("form")).getByRole("button", { name: "Sign in" }),
    );

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining("/auth/login"),
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({
            email: "test@example.com",
            password: "password12345",
          }),
        })
      );
    });
  });

  it("shows a safe message when login fails", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/auth/login")) {
        return new Response(
          JSON.stringify({
            error: { code: "invalid_credentials", message: "bad" },
          }),
          { status: 401, headers: { "Content-Type": "application/json" } }
        );
      }
      return new Response("{}", { status: 404 });
    });
    globalThis.fetch = fetchMock as typeof fetch;

    renderWithProviders(<LoginPage />);
    await waitFor(() => {
      expect(screen.getByLabelText("Email")).toBeInTheDocument();
    });
    await userEvent.type(screen.getByLabelText("Email"), "test@example.com");
    await userEvent.type(screen.getByLabelText("Password"), "wrongwrongwrong");
    await userEvent.click(
      within(screen.getByRole("form")).getByRole("button", { name: "Sign in" }),
    );

    await waitFor(() => {
      expect(
        screen.getByText("Invalid email or password. Please try again.")
      ).toBeInTheDocument();
    });
  });
});
