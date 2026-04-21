import {
  mergeParamsWithConnectorInstance,
  parseAvailableInstances,
} from "../src/util/connectorInstance.js";

describe("mergeParamsWithConnectorInstance", () => {
  it("merges connector_instance into a params object", () => {
    expect(
      mergeParamsWithConnectorInstance({ channel: "#general" }, "550e8400-e29b-41d4-a716-446655440000"),
    ).toEqual({
      channel: "#general",
      connector_instance: "550e8400-e29b-41d4-a716-446655440000",
    });
  });

  it("rejects non-object params", () => {
    expect(() => mergeParamsWithConnectorInstance([], "x")).toThrow(
      "--params must be a JSON object when using --instance",
    );
  });
});

describe("parseAvailableInstances", () => {
  it("parses object-shaped entries", () => {
    const list = parseAvailableInstances({
      available_instances: [
        { id: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", display_name: "Work" },
        { id: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb" },
      ],
    });
    expect(list).toEqual([
      { id: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", display_name: "Work" },
      { id: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb" },
    ]);
  });

  it("ignores legacy string nicknames (no UUID)", () => {
    const list = parseAvailableInstances({
      available_instances: ["Alpha", "Beta"],
    });
    expect(list).toEqual([]);
  });

  it("accepts legacy string UUIDs only", () => {
    const list = parseAvailableInstances({
      available_instances: ["aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"],
    });
    expect(list).toEqual([{ id: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" }]);
  });
});
