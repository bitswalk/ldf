import { describe, it, expect, beforeEach } from "vitest";
import {
  getBrandingAssetURL,
  DEFAULT_APP_NAME,
  APP_NAME_MAX_LENGTH,
} from "./branding";
import { setServerUrl } from "./storage";

beforeEach(() => {
  localStorage.clear();
});

describe("getBrandingAssetURL", () => {
  it("returns null when no server URL is set", () => {
    expect(getBrandingAssetURL("logo")).toBeNull();
  });

  it("constructs correct URL for logo", () => {
    setServerUrl("https://example.com");
    expect(getBrandingAssetURL("logo")).toBe(
      "https://example.com/v1/branding/logo",
    );
  });

  it("constructs correct URL for favicon", () => {
    setServerUrl("https://example.com");
    expect(getBrandingAssetURL("favicon")).toBe(
      "https://example.com/v1/branding/favicon",
    );
  });
});

describe("Branding constants", () => {
  it("DEFAULT_APP_NAME is defined", () => {
    expect(DEFAULT_APP_NAME).toBe("Linux Distribution Factory");
  });

  it("APP_NAME_MAX_LENGTH is 32", () => {
    expect(APP_NAME_MAX_LENGTH).toBe(32);
  });
});
