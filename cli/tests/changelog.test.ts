import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import {
  compareVersions,
  currentCliVersion,
  entriesSinceVersion,
  markChangelogRead,
  parseChangelog,
  getLastChangelogVersion,
} from "../src/changelog.js";

const SAMPLE = `# Changelog

## [0.2.0] - 2026-06-01

### Added
- New feature

## [0.1.0] - 2026-01-01

### Added
- Initial
`;

describe("changelog", () => {
  it("parses version sections", () => {
    const entries = parseChangelog(SAMPLE);
    expect(entries).toHaveLength(2);
    expect(entries[0]?.version).toBe("0.2.0");
  });

  it("compareVersions orders semver", () => {
    expect(compareVersions("0.2.0", "0.1.0")).toBeGreaterThan(0);
    expect(compareVersions("0.1.0", "0.2.0")).toBeLessThan(0);
  });

  it("entriesSinceVersion filters", () => {
    const entries = parseChangelog(SAMPLE);
    const unread = entriesSinceVersion(entries, "0.1.0");
    expect(unread).toHaveLength(1);
    expect(unread[0]?.version).toBe("0.2.0");
  });

  it("currentCliVersion reads package.json", () => {
    expect(currentCliVersion()).toMatch(/^\d+\.\d+\.\d+$/);
  });

  it("markChangelogRead persists version", () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "ps-changelog-"));
    process.env["PS_CLI_TEST_CONFIG_DIR"] = tmp;
    markChangelogRead("1.2.3");
    expect(getLastChangelogVersion()).toBe("1.2.3");
    delete process.env["PS_CLI_TEST_CONFIG_DIR"];
  });
});
