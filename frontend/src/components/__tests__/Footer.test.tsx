import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { Footer } from "../Footer";

describe("Footer", () => {
  it("renders the web release version", () => {
    vi.stubEnv("VITE_WEB_VERSION", "1.2.3");

    render(
      <MemoryRouter>
        <Footer />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("app-version")).toHaveTextContent("v1.2.3");
  });
});
