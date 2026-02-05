import { describe, it, expect } from "vitest";
import {
  parseVersionFilter,
  matchesFilter,
  filterVersions,
  filterVersionsWithReasons,
} from "./globFilter";

describe("parseVersionFilter", () => {
  it("returns empty array for empty string", () => {
    expect(parseVersionFilter("")).toEqual([]);
  });

  it("returns empty array for whitespace-only string", () => {
    expect(parseVersionFilter("   ")).toEqual([]);
  });

  it("parses a single inclusion pattern", () => {
    const result = parseVersionFilter("v1.*");
    expect(result).toHaveLength(1);
    expect(result[0].isExclusion).toBe(false);
    expect(result[0].pattern).toBe("v1.*");
  });

  it("parses a single exclusion pattern", () => {
    const result = parseVersionFilter("!*-rc*");
    expect(result).toHaveLength(1);
    expect(result[0].isExclusion).toBe(true);
    expect(result[0].pattern).toBe("!*-rc*");
  });

  it("parses comma-separated mixed patterns", () => {
    const result = parseVersionFilter("v1.*, !*-rc*, v2.0");
    expect(result).toHaveLength(3);
    expect(result[0].isExclusion).toBe(false);
    expect(result[1].isExclusion).toBe(true);
    expect(result[2].isExclusion).toBe(false);
  });

  it("trims whitespace from patterns", () => {
    const result = parseVersionFilter("  v1.*  ,  !*-rc*  ");
    expect(result).toHaveLength(2);
    expect(result[0].pattern).toBe("v1.*");
    expect(result[1].pattern).toBe("!*-rc*");
  });
});

describe("matchesFilter", () => {
  it("returns true when no patterns are provided", () => {
    expect(matchesFilter("v1.0", [])).toBe(true);
  });

  it("matches inclusion patterns", () => {
    const patterns = parseVersionFilter("v1.*");
    expect(matchesFilter("v1.0", patterns)).toBe(true);
    expect(matchesFilter("v1.5.3", patterns)).toBe(true);
    expect(matchesFilter("v2.0", patterns)).toBe(false);
  });

  it("matches exclusion patterns", () => {
    const patterns = parseVersionFilter("!*-rc*");
    expect(matchesFilter("v1.0", patterns)).toBe(true);
    expect(matchesFilter("v1.0-rc1", patterns)).toBe(false);
  });

  it("exclusion takes precedence over inclusion", () => {
    const patterns = parseVersionFilter("v1.*, !v1.0-rc*");
    expect(matchesFilter("v1.0", patterns)).toBe(true);
    expect(matchesFilter("v1.0-rc1", patterns)).toBe(false);
  });

  it("case insensitive matching", () => {
    const patterns = parseVersionFilter("V1.*");
    expect(matchesFilter("v1.0", patterns)).toBe(true);
  });

  it("with only exclusion patterns, includes by default", () => {
    const patterns = parseVersionFilter("!*-beta*");
    expect(matchesFilter("v1.0", patterns)).toBe(true);
    expect(matchesFilter("v2.0-beta", patterns)).toBe(false);
  });

  it("? matches single character", () => {
    const patterns = parseVersionFilter("v1.?");
    expect(matchesFilter("v1.0", patterns)).toBe(true);
    expect(matchesFilter("v1.5", patterns)).toBe(true);
    expect(matchesFilter("v1.10", patterns)).toBe(false);
  });
});

describe("filterVersions", () => {
  const versions = ["v1.0", "v1.1", "v1.1-rc1", "v2.0", "v2.0-beta"];

  it("returns all versions with empty filter", () => {
    expect(filterVersions(versions, "")).toEqual(versions);
  });

  it("filters to matching versions", () => {
    expect(filterVersions(versions, "v1.*")).toEqual([
      "v1.0",
      "v1.1",
      "v1.1-rc1",
    ]);
  });

  it("excludes matching versions", () => {
    expect(filterVersions(versions, "!*-rc*, !*-beta*")).toEqual([
      "v1.0",
      "v1.1",
      "v2.0",
    ]);
  });

  it("combines inclusion and exclusion", () => {
    expect(filterVersions(versions, "v1.*, !*-rc*")).toEqual([
      "v1.0",
      "v1.1",
    ]);
  });
});

describe("filterVersionsWithReasons", () => {
  it("includes all versions with empty filter and no reason", () => {
    const results = filterVersionsWithReasons(["v1.0", "v2.0"], "");
    expect(results).toEqual([
      { version: "v1.0", included: true },
      { version: "v2.0", included: true },
    ]);
  });

  it("provides exclusion reason", () => {
    const results = filterVersionsWithReasons(["v1.0-rc1"], "!*-rc*");
    expect(results[0].included).toBe(false);
    expect(results[0].reason).toBe("excluded by !*-rc*");
  });

  it("provides inclusion reason", () => {
    const results = filterVersionsWithReasons(["v1.0"], "v1.*");
    expect(results[0].included).toBe(true);
    expect(results[0].reason).toBe("matched v1.*");
  });

  it("provides no-match reason for unmatched versions", () => {
    const results = filterVersionsWithReasons(["v2.0"], "v1.*");
    expect(results[0].included).toBe(false);
    expect(results[0].reason).toBe("no inclusion pattern matched");
  });
});
