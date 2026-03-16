import type { Meta, StoryObj } from "@storybook/react";

const meta: Meta = {
  title: "Components/SchemaParameterDetails/Value Style Variants",
  parameters: { layout: "centered" },
};
export default meta;

const singleLineValue = "alice@example.com";
const multilineValue = `Hi Alice,

Here are this week's standup notes.

The team made great progress on the API migration. All endpoints are now versioned and the legacy routes have been deprecated with a 90-day sunset window.

The new dashboard is ready for QA — please review it before Thursday's release window.

Best,
Chiedobot`;

type VariantStyle = { bg: string; border: string; color: string };

function VariantBlock({ label, style }: { label: string; style: VariantStyle }) {
  const blockStyle = { background: style.bg, border: `1px solid ${style.border}`, color: style.color };
  return (
    <div className="space-y-4">
      <p className="text-xs font-bold uppercase tracking-widest text-muted-foreground border-b pb-1">{label}</p>
      {/* Single line */}
      <div className="space-y-1.5">
        <p className="text-foreground/70 text-xs font-semibold tracking-wide uppercase">Recipient Email Address</p>
        <span
          className="inline-block rounded-md px-2.5 py-1 text-sm font-medium break-all"
          style={blockStyle}
        >
          {singleLineValue}
        </span>
      </div>
      {/* Multiline */}
      <div className="space-y-1.5">
        <p className="text-foreground/70 text-xs font-semibold tracking-wide uppercase">Email Body</p>
        <pre
          className="rounded-md px-3 py-2 text-xs leading-relaxed whitespace-pre-wrap break-words font-sans"
          style={blockStyle}
        >
          {multilineValue}
        </pre>
      </div>
    </div>
  );
}

function AllVariants() {
  return (
    <div className="w-[500px] space-y-8">
      <VariantBlock
        label="A — Slate-100 bg, slate-700 text, slate-200 border"
        style={{ bg: "#f1f5f9", border: "#e2e8f0", color: "#334155" }}
      />
      <VariantBlock
        label="B — Slate-100 bg, slate-900 text, slate-300 border"
        style={{ bg: "#f1f5f9", border: "#cbd5e1", color: "#0f172a" }}
      />
      <VariantBlock
        label="C — Slate-50 bg, slate-600 text, slate-200 border"
        style={{ bg: "#f8fafc", border: "#e2e8f0", color: "#475569" }}
      />
      <VariantBlock
        label="D — Zinc-100 bg, zinc-700 text, zinc-200 border"
        style={{ bg: "#f4f4f5", border: "#e4e4e7", color: "#3f3f46" }}
      />
      <VariantBlock
        label="E — Zinc-100 bg, zinc-900 text, zinc-300 border"
        style={{ bg: "#f4f4f5", border: "#d4d4d8", color: "#18181b" }}
      />
    </div>
  );
}

export const Variants: StoryObj = {
  render: () => <AllVariants />,
  decorators: [
    (Story) => (
      <div className="bg-background p-6 rounded-lg border">
        <Story />
      </div>
    ),
  ],
};
