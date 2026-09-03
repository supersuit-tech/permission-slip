import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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
  it("renders humanized keys as labels, not schema descriptions", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets" }}
        schema={schema}
      />,
    );

    expect(screen.getByText("Owner")).toBeInTheDocument();
    expect(screen.getByText("Repo")).toBeInTheDocument();
    expect(screen.queryByText("Repository owner")).not.toBeInTheDocument();
  });

  it("uses x-ui label when available", () => {
    const labeledSchema: ParametersSchema = {
      type: "object",
      properties: {
        message_id: {
          type: "string",
          description: "Stable IMAP UID of a single email to archive",
          "x-ui": { label: "Message" },
        },
      },
    };

    render(
      <SchemaParameterDetails
        parameters={{ message_id: "12345" }}
        schema={labeledSchema}
      />,
    );

    expect(screen.getByText("Message")).toBeInTheDocument();
    expect(
      screen.queryByText("Stable IMAP UID of a single email to archive"),
    ).not.toBeInTheDocument();
  });

  it("renders parameter values in the clean view", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets" }}
        schema={schema}
      />,
    );

    expect(screen.getByText("acme")).toBeInTheDocument();
    expect(screen.getByText("widgets")).toBeInTheDocument();
  });

  it("hides developer chrome from the default view", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets", merge_method: "squash" }}
        schema={schema}
      />,
    );

    expect(screen.queryByText("owner")).not.toBeInTheDocument();
    expect(screen.queryByText("one of: merge, squash, rebase")).not.toBeInTheDocument();
    expect(screen.queryByText("default")).not.toBeInTheDocument();
  });

  it("shows full schema details behind the developer toggle", async () => {
    const user = userEvent.setup();

    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme" }}
        schema={schema}
      />,
    );

    await user.click(screen.getByRole("button", { name: /developer details/i }));

    expect(screen.getByText("Repository owner")).toBeInTheDocument();
    expect(screen.getByText("owner")).toBeInTheDocument();
    expect(screen.getByText("missing")).toBeInTheDocument();
  });

  it("shows enum options in developer details", async () => {
    const user = userEvent.setup();

    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets", merge_method: "squash" }}
        schema={schema}
      />,
    );

    await user.click(screen.getByRole("button", { name: /developer details/i }));

    expect(screen.getByText("one of: merge, squash, rebase")).toBeInTheDocument();
  });

  it("shows default badge in developer details", async () => {
    const user = userEvent.setup();

    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets", merge_method: "merge" }}
        schema={schema}
      />,
    );

    await user.click(screen.getByRole("button", { name: /developer details/i }));

    expect(screen.getByText("default")).toBeInTheDocument();
  });

  it("hides unprovided optional parameters from the clean view", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets" }}
        schema={schema}
      />,
    );

    expect(screen.queryByText("Title")).not.toBeInTheDocument();
    expect(screen.queryByText("not provided")).not.toBeInTheDocument();
  });

  it("renders extra parameters not in schema with humanized keys", () => {
    render(
      <SchemaParameterDetails
        parameters={{ owner: "acme", repo: "widgets", custom_field: "hello" }}
        schema={schema}
      />,
    );

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

  it("shows resolved Slack channel name with raw ID for slack.send_message", () => {
    const slackSchema: ParametersSchema = {
      type: "object",
      required: ["channel", "message"],
      properties: {
        channel: { type: "string", description: "Channel" },
        message: { type: "string", description: "Message" },
      },
    };
    render(
      <SchemaParameterDetails
        parameters={{ channel: "C0123", message: "hi" }}
        schema={slackSchema}
        actionType="slack.send_message"
        resourceDetails={{ channel_name: "#general" }}
      />,
    );

    expect(screen.getByText("#general (C0123)")).toBeInTheDocument();
  });

  it("shows resolved Drive folder name with raw ID", () => {
    const driveSchema: ParametersSchema = {
      type: "object",
      required: ["name"],
      properties: {
        name: { type: "string", description: "File name" },
        folder_id: { type: "string", description: "Folder ID", "x-ui": { label: "Folder" } },
      },
    };
    render(
      <SchemaParameterDetails
        parameters={{ name: "notes.pdf", folder_id: "0AKbIIKZ8knmBUk9PVA" }}
        schema={driveSchema}
        resourceDetails={{ folder_name: "Finance Shared Drive" }}
      />,
    );

    expect(screen.getByText("Finance Shared Drive (0AKbIIKZ8knmBUk9PVA)")).toBeInTheDocument();
  });

  it("shows nested Shared Drive folder with the drive title", () => {
    const driveSchema: ParametersSchema = {
      type: "object",
      required: ["name"],
      properties: {
        name: { type: "string", description: "File name" },
        folder_id: { type: "string", description: "Folder ID", "x-ui": { label: "Folder" } },
      },
    };
    render(
      <SchemaParameterDetails
        parameters={{ name: "ps-ui-test.pdf", folder_id: "1Xv2Naa6LjElcSK55wb9HigrLrAaYPE0d" }}
        schema={driveSchema}
        resourceDetails={{ folder_name: "2026-documents in Chiedo's assistant drive" }}
      />,
    );

    expect(
      screen.getByText("2026-documents in Chiedo's assistant drive (1Xv2Naa6LjElcSK55wb9HigrLrAaYPE0d)"),
    ).toBeInTheDocument();
  });

  it("summarizes content_base64 instead of dumping encoded bytes", () => {
    const driveSchema: ParametersSchema = {
      type: "object",
      properties: {
        name: { type: "string" },
        content_base64: { type: "string", "x-ui": { label: "File" } },
        mime_type: { type: "string" },
      },
    };
    const pdf = "JVBERi0xLjQK";
    render(
      <SchemaParameterDetails
        parameters={{
          name: "receipt.pdf",
          content_base64: pdf,
          mime_type: "application/pdf",
        }}
        schema={driveSchema}
      />,
    );

    expect(screen.getByText(/PDF\s\u00b7/)).toBeInTheDocument();
    expect(screen.queryByText(pdf)).not.toBeInTheDocument();
  });
});
