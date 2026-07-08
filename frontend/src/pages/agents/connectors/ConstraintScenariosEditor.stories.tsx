import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { ConstraintScenariosEditor } from "@/pages/agents/connectors/ConstraintScenariosEditor";
import {
  constraintsToFormState,
  emptyConstraintRow,
} from "@/lib/structuredConstraints";
import type { ParametersSchema } from "@/lib/parameterSchema";

const SLACK_SCHEMA: ParametersSchema = {
  type: "object",
  required: ["channel"],
  properties: {
    channel: { type: "string", description: "Slack channel" },
    text: { type: "string", description: "Message text" },
  },
};

function ScenariosEditorDemo({
  initialNegation = false,
  initialMultiScenario = false,
}: {
  initialNegation?: boolean;
  initialMultiScenario?: boolean;
}) {
  const [form, setForm] = useState(() => {
    if (initialMultiScenario) {
      return constraintsToFormState({
        $version: 2,
        match: "any",
        groups: [
          {
            match: "all",
            conditions: [
              { field: "channel", op: "matches", value: "#engineering" },
            ],
          },
          {
            match: "all",
            conditions: [
              { field: "channel", op: "matches", value: "#incidents" },
            ],
          },
        ],
      });
    }
    const base = constraintsToFormState(null);
    const scenario = base.scenarios[0];
    if (scenario && initialNegation) {
      scenario.paramRows.channel = [
        { ...emptyConstraintRow(), value: "#engineering", mode: "fixed" },
        { ...emptyConstraintRow(), value: "#releases", mode: "fixed" },
        {
          ...emptyConstraintRow("does_not_match"),
          value: "#executive-only",
          mode: "fixed",
        },
      ];
    }
    return base;
  });

  return (
    <div className="max-w-lg rounded-lg border bg-background p-4">
      <ConstraintScenariosEditor
        form={form}
        onChange={setForm}
        parametersSchema={SLACK_SCHEMA}
        metaFields={["from", "to"]}
      />
    </div>
  );
}

const meta: Meta<typeof ScenariosEditorDemo> = {
  title: "Connectors/ConstraintScenariosEditor",
  component: ScenariosEditorDemo,
  parameters: { layout: "centered" },
};

export default meta;

type Story = StoryObj<typeof ScenariosEditorDemo>;

export const Default: Story = {};

export const WithNegation: Story = {
  args: { initialNegation: true },
  name: "Allow/deny rows",
};

export const MultiScenario: Story = {
  args: { initialMultiScenario: true },
  name: "Multiple scenarios",
};

const IMESSAGE_SCHEMA: ParametersSchema = {
  type: "object",
  properties: {
    sort: { type: "string", enum: ["desc", "asc"] },
    order_by: {
      type: "string",
      enum: ["last_activity", "contact_name"],
    },
    unread_only: { type: "boolean" },
  },
};

function IMessageConstraintsDemo() {
  const [form, setForm] = useState(() => {
    const base = constraintsToFormState(null);
    const scenario = base.scenarios[0];
    if (scenario) {
      scenario.paramRows.sort = [
        { ...emptyConstraintRow(), mode: "wildcard", value: "*" },
      ];
      scenario.paramRows.order_by = [
        { ...emptyConstraintRow(), value: "last_activity", mode: "fixed" },
      ];
      scenario.paramRows.unread_only = [
        { ...emptyConstraintRow(), value: "true", mode: "fixed" },
      ];
    }
    return base;
  });

  return (
    <div className="max-w-lg rounded-lg border bg-background p-4">
      <ConstraintScenariosEditor
        form={form}
        onChange={setForm}
        parametersSchema={IMESSAGE_SCHEMA}
        metaFields={[]}
      />
    </div>
  );
}

export const EnumAndBooleanFields: StoryObj<typeof IMessageConstraintsDemo> = {
  render: () => <IMessageConstraintsDemo />,
  name: "Enum and boolean fields",
};
