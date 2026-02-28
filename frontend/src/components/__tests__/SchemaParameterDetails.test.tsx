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

  it("shows type annotations for each schema property", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme" }}
        schema={schema}
      />,
    );

    // All four schema properties have type "string"
    const typeAnnotations = screen.getAllByText("string");
    expect(typeAnnotations.length).toBe(4);
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

  it("shows 'not provided' for optional parameters not provided", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets" }}
        schema={schema}
      />,
    );

    // "title" and "merge_method" are both optional and not provided
    const notProvided = screen.getAllByText("not provided");
    expect(notProvided.length).toBe(2);
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

  it("renders extra parameters not in schema", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets", custom_field: "hello" }}
        schema={schema}
      />,
    );

    expect(screen.getByText("custom_field")).toBeInTheDocument();
    expect(screen.getByText("hello")).toBeInTheDocument();
  });

  it("falls back to key-value display when schema is null", () => {
    render(
      <SchemaParameterDetails
        parameters={{ foo: "bar", count: "42" }}
        schema={null}
      />,
    );

    expect(screen.getByText("foo")).toBeInTheDocument();
    expect(screen.getByText("bar")).toBeInTheDocument();
    expect(screen.getByText("count")).toBeInTheDocument();
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

    expect(screen.getByText("key")).toBeInTheDocument();
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
