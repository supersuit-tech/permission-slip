import { parseConstraintsJson } from "../src/formatStandingApprovalConstraints.js";

describe("parseConstraintsJson", () => {
  it("parses a JSON object", () => {
    expect(parseConstraintsJson('{"recipient":"*@acme.com"}')).toEqual({
      recipient: "*@acme.com",
    });
  });

  it("rejects invalid JSON", () => {
    expect(() => parseConstraintsJson("not-json")).toThrow(
      "--constraints must be valid JSON",
    );
  });

  it("rejects non-object values", () => {
    expect(() => parseConstraintsJson('["a"]')).toThrow(
      "--constraints must be a JSON object",
    );
  });
});
