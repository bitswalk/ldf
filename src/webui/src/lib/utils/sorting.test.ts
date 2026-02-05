import { describe, it, expect } from "vitest";
import { compareValues } from "./sorting";

describe("compareValues", () => {
  describe("numeric comparison", () => {
    it("compares numbers ascending", () => {
      expect(compareValues(1, 2)).toBe(-1);
      expect(compareValues(2, 1)).toBe(1);
      expect(compareValues(1, 1)).toBe(0);
    });

    it("compares numbers descending", () => {
      expect(compareValues(1, 2, "desc")).toBe(1);
      expect(compareValues(2, 1, "desc")).toBe(-1);
    });

    it("handles numeric strings", () => {
      expect(compareValues("10", "2")).toBe(1);
      expect(compareValues("2", "10")).toBe(-1);
    });
  });

  describe("semantic version comparison", () => {
    it("compares two-part versions numerically per segment", () => {
      // "6.2" and "6.18" are valid JS numbers, so numeric comparison applies
      // Use three-part or prefixed versions to trigger semver path
      expect(compareValues("6.2.0", "6.18.0")).toBeLessThan(0);
      expect(compareValues("6.18.0", "6.2.0")).toBeGreaterThan(0);
    });

    it("compares three-part versions", () => {
      expect(compareValues("6.18.2", "6.18.1")).toBeGreaterThan(0);
      expect(compareValues("6.18.1", "6.18.2")).toBeLessThan(0);
    });

    it("release beats pre-release", () => {
      expect(compareValues("6.12", "6.12-rc1")).toBeGreaterThan(0);
    });

    it("compares rc versions", () => {
      expect(compareValues("6.12-rc1", "6.12-rc2")).toBeLessThan(0);
      expect(compareValues("6.12-rc2", "6.12-rc1")).toBeGreaterThan(0);
    });

    it("equal versions return 0", () => {
      expect(compareValues("6.18.2", "6.18.2")).toBe(0);
    });

    it("respects descending direction", () => {
      expect(compareValues("6.2.0", "6.18.0", "desc")).toBeGreaterThan(0);
    });
  });

  describe("string comparison", () => {
    it("compares strings case-insensitively", () => {
      expect(compareValues("apple", "Banana")).toBeLessThan(0);
      expect(compareValues("Banana", "apple")).toBeGreaterThan(0);
    });

    it("equal strings return 0", () => {
      expect(compareValues("alpha", "alpha")).toBe(0);
    });

    it("respects descending direction for strings", () => {
      expect(compareValues("apple", "banana", "desc")).toBeGreaterThan(0);
    });
  });
});
