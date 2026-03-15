import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { SchemaParameterDetails } from "../SchemaParameterDetails";
import type { ParametersSchema } from "@/lib/parameterSchema";

const schema: ParametersSchema = {
  type: "object",
  required: ["owner", "repo"],
  properties: {
    owner: { type: "string", description: "Repository owner" },
    repo: { type: "string", description: "Repository name" },
    title: { type: "string", description: "Issue title" },
    merge_method: {
      type: "string",
      description: "Merge strategy",
      enum: ["merge", "squash", "rebase"],
      default: "merge",
    },
  },
};

describe("SchemaParameterDetails", () => {
  it("renders parameter descriptions as labels", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets" }}
        schema={schema}
      />,
    );

    expect(screen.getByText("Repository owner")).toBeInTheDocument();
    expect(screen.getByText("Repository name")).toBeInTheDocument();
  });

  it("renders parameter values", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets" }}
        schema={schema}
      />,
    );

    expect(screen.getByText("acme")).toBeInTheDocument();
    expect(screen.getByText("widgets")).toBeInTheDocument();
  });

  it("shows parameter key as secondary label when description exists", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme" }}
        schema={schema}
      />,
    );

    // The key is shown as a small code element alongside the description label
    expect(screen.getByText("owner")).toBeInTheDocument();
  });

  it("shows type annotations for provided parameters", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets" }}
        schema={schema}
      />,
    );

    // Only provided params + required missing params show type annotations.
    // owner and repo are provided; title and merge_method are optional and hidden.
    const typeAnnotations = screen.getAllByText("string");
    expect(typeAnnotations.length).toBe(2);
  });

  it("shows 'missing' badge for required parameters not provided", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme" }}
        schema={schema}
      />,
    );

    // "repo" is required but not provided
    expect(screen.getByText("missing")).toBeInTheDocument();
  });

  it("hides unprovided optional parameters", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets" }}
        schema={schema}
      />,
    );

    // "title" and "merge_method" are optional and not provided — should be hidden
    expect(screen.queryByText("not provided")).not.toBeInTheDocument();
    expect(screen.queryByText("Issue title")).not.toBeInTheDocument();
  });

  it("shows enum options for enum parameters", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets", merge_method: "squash" }}
        schema={schema}
      />,
    );

    expect(screen.getByText("(merge | squash | rebase)")).toBeInTheDocument();
  });

  it("shows 'default' badge when value matches default", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets", merge_method: "merge" }}
        schema={schema}
      />,
    );

    expect(screen.getByText("default")).toBeInTheDocument();
  });

  it("renders extra parameters not in schema with humanized keys", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets", custom_field: "hello" }}
        schema={schema}
      />,
    );

    // humanizeKey turns "custom_field" into "Custom field"
    expect(screen.getByText("Custom field")).toBeInTheDocument();
    expect(screen.getByText("hello")).toBeInTheDocument();
  });

  it("falls back to key-value display when schema is null", () => {
    render(
      <SchemaParameterDetails
        parameters={{ foo: "bar", count: "42" }}
        schema={null}
      />,
    );

    // humanizeKey capitalizes the first letter
    expect(screen.getByText("Foo")).toBeInTheDocument();
    expect(screen.getByText("bar")).toBeInTheDocument();
    expect(screen.getByText("Count")).toBeInTheDocument();
    expect(screen.getByText("42")).toBeInTheDocument();
  });

  it("falls back to key-value display when schema has no properties", () => {
    const emptySchema: ParametersSchema = { type: "object" };

    render(
      <SchemaParameterDetails
        parameters={{ key: "value" }}
        schema={emptySchema}
      />,
    );

    expect(screen.getByText("Key")).toBeInTheDocument();
    expect(screen.getByText("value")).toBeInTheDocument();
  });

  it("formats array values as comma-separated", () => {
    render(
      <SchemaParameterDetails
        parameters={{ labels: ["bug", "urgent"] }}
        schema={null}
      />,
    );

    expect(screen.getByText("bug, urgent")).toBeInTheDocument();
  });

  it("formats null/undefined values as em dash", () => {
    render(
      <SchemaParameterDetails
        parameters={{ empty: null }}
        schema={null}
      />,
    );

    expect(screen.getByText("\u2014")).toBeInTheDocument();
  });
});
