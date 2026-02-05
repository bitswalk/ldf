import { describe, it, expect } from "vitest";
import { getForgeTypeDisplayName, FORGE_TYPES } from "./forge";
import type { ForgeType } from "./forge";

describe("getForgeTypeDisplayName", () => {
  const cases: [ForgeType, string][] = [
    ["github", "GitHub"],
    ["gitlab", "GitLab"],
    ["gitea", "Gitea"],
    ["codeberg", "Codeberg"],
    ["forgejo", "Forgejo"],
    ["generic", "Generic"],
  ];

  it.each(cases)("maps '%s' to '%s'", (forgeType, expected) => {
    expect(getForgeTypeDisplayName(forgeType)).toBe(expected);
  });

  it("returns the raw value for unknown forge types", () => {
    expect(getForgeTypeDisplayName("unknown" as ForgeType)).toBe("unknown");
  });
});

describe("FORGE_TYPES", () => {
  it("contains all six forge types", () => {
    expect(FORGE_TYPES).toHaveLength(6);
  });

  it("each entry has type, display_name, and description", () => {
    for (const ft of FORGE_TYPES) {
      expect(ft).toHaveProperty("type");
      expect(ft).toHaveProperty("display_name");
      expect(ft).toHaveProperty("description");
      expect(typeof ft.type).toBe("string");
      expect(typeof ft.display_name).toBe("string");
      expect(typeof ft.description).toBe("string");
    }
  });

  it("display names match getForgeTypeDisplayName", () => {
    for (const ft of FORGE_TYPES) {
      expect(ft.display_name).toBe(getForgeTypeDisplayName(ft.type));
    }
  });
});
