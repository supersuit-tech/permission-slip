import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { parseProtonBatchEmails } from "../protonBatchEmailsUtils";
import { ProtonBatchEmailsCard } from "../ProtonBatchEmailsCard";

const BATCH_DETAILS = {
  messages: {
    "232": {
      subject: "Second email",
      from: ["b@example.com"],
      to: ["me@proton.me"],
      date: "2026-03-02T09:00:00Z",
    },
    "231": {
      subject: "First email",
      from: ["a@example.com"],
      to: ["me@proton.me"],
      date: "2026-03-01T09:00:00Z",
    },
  },
};

describe("parseProtonBatchEmails", () => {
  it("returns null for missing or invalid messages", () => {
    expect(parseProtonBatchEmails(undefined)).toBeNull();
    expect(parseProtonBatchEmails({})).toBeNull();
    expect(parseProtonBatchEmails({ messages: "bad" })).toBeNull();
    expect(parseProtonBatchEmails({ messages: [] })).toBeNull();
  });

  it("returns null for a single resolved message (flat fields cover it)", () => {
    expect(
      parseProtonBatchEmails({ messages: { "10": { subject: "Only one" } } }),
    ).toBeNull();
  });

  it("parses and sorts batch messages by UID", () => {
    const parsed = parseProtonBatchEmails(BATCH_DETAILS);
    expect(parsed).toHaveLength(2);
    expect(parsed?.[0]?.uid).toBe("231");
    expect(parsed?.[0]?.subject).toBe("First email");
    expect(parsed?.[1]?.uid).toBe("232");
  });
});

describe("ProtonBatchEmailsCard", () => {
  let renderer: ReactTestRenderer;

  afterEach(async () => {
    await act(async () => {
      renderer?.unmount();
    });
  });

  function findByTestId(testID: string) {
    return renderer.root.find((node) => node.props.testID === testID);
  }

  function hasTestId(testID: string) {
    try {
      findByTestId(testID);
      return true;
    } catch {
      return false;
    }
  }

  async function renderCard() {
    const emails = parseProtonBatchEmails(BATCH_DETAILS);
    if (!emails) throw new Error("expected batch emails");
    await act(async () => {
      renderer = create(createElement(ProtonBatchEmailsCard, { emails }));
    });
  }

  it("renders one collapsed row per email", async () => {
    await renderCard();
    expect(hasTestId("batch-email-toggle-231")).toBe(true);
    expect(hasTestId("batch-email-toggle-232")).toBe(true);
    expect(hasTestId("batch-email-details-231")).toBe(false);
    expect(hasTestId("batch-email-details-232")).toBe(false);
  });

  it("expands and collapses a row's details independently", async () => {
    await renderCard();

    await act(async () => {
      findByTestId("batch-email-toggle-231").props.onPress();
    });
    expect(hasTestId("batch-email-details-231")).toBe(true);
    expect(hasTestId("batch-email-details-232")).toBe(false);

    await act(async () => {
      findByTestId("batch-email-toggle-231").props.onPress();
    });
    expect(hasTestId("batch-email-details-231")).toBe(false);
  });
});
