import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { Avatar, getInitials } from "../ui/avatar";

describe("getInitials", () => {
  it("returns first letters of first and last name", () => {
    expect(getInitials("John Doe")).toBe("JD");
  });

  it("returns first letter for single-word name", () => {
    expect(getInitials("Alice")).toBe("A");
  });

  it("uses first and last name for multi-word names", () => {
    expect(getInitials("Mary Jane Watson")).toBe("MW");
  });

  it("falls back to email first letter when no name", () => {
    expect(getInitials(null, "alice@example.com")).toBe("A");
  });

  it("returns ? when no name or email", () => {
    expect(getInitials(null, null)).toBe("?");
  });

  it("prefers name over email", () => {
    expect(getInitials("Bob", "alice@example.com")).toBe("B");
  });

  it("falls back to email when name is empty string", () => {
    expect(getInitials("", "alice@example.com")).toBe("A");
  });

  it("falls back to email when name is whitespace", () => {
    expect(getInitials("   ", "alice@example.com")).toBe("A");
  });

  it("returns ? when email is empty string", () => {
    expect(getInitials(null, "")).toBe("?");
  });

  it("returns ? when email is whitespace", () => {
    expect(getInitials(null, "   ")).toBe("?");
  });
});

describe("Avatar", () => {
  it("renders initials from email", () => {
    render(<Avatar email="test@example.com" />);
    expect(screen.getByText("T")).toBeInTheDocument();
  });

  it("renders initials from name", () => {
    render(<Avatar name="Jane Doe" email="test@example.com" />);
    expect(screen.getByText("JD")).toBeInTheDocument();
  });

  it("renders as a button by default", () => {
    render(<Avatar email="test@example.com" />);
    expect(screen.getByRole("button")).toBeInTheDocument();
  });

  it("renders as a span when as='span'", () => {
    const { container } = render(<Avatar as="span" email="test@example.com" />);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(container.querySelector("span[data-slot='avatar']")).toBeInTheDocument();
    expect(screen.getByText("T")).toBeInTheDocument();
  });
});
