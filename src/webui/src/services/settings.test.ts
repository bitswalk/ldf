import { describe, it, expect, beforeEach } from "vitest";
import { isRootUser, groupSettingsByCategory } from "./settings";
import type { ServerSetting } from "./settings";
import { setUserInfo } from "./storage";

beforeEach(() => {
  localStorage.clear();
});

describe("isRootUser", () => {
  it("returns false when no user info is stored", () => {
    expect(isRootUser()).toBe(false);
  });

  it("returns true when user role is root", () => {
    setUserInfo({ id: "1", name: "admin", email: "a@b.c", role: "root" });
    expect(isRootUser()).toBe(true);
  });

  it("returns false when user role is developer", () => {
    setUserInfo({ id: "1", name: "dev", email: "a@b.c", role: "developer" });
    expect(isRootUser()).toBe(false);
  });

  it("returns false when user role is anonymous", () => {
    setUserInfo({ id: "1", name: "anon", email: "a@b.c", role: "anonymous" });
    expect(isRootUser()).toBe(false);
  });
});

describe("groupSettingsByCategory", () => {
  const mockSettings: ServerSetting[] = [
    {
      key: "server.port",
      value: 8080,
      type: "int",
      description: "Port",
      rebootRequired: true,
      category: "server",
    },
    {
      key: "log.level",
      value: "info",
      type: "string",
      description: "Log level",
      rebootRequired: false,
      category: "log",
    },
    {
      key: "server.host",
      value: "0.0.0.0",
      type: "string",
      description: "Host",
      rebootRequired: true,
      category: "server",
    },
    {
      key: "webui.devmode",
      value: false,
      type: "bool",
      description: "Dev mode",
      rebootRequired: false,
      category: "webui",
    },
  ];

  it("groups settings by category", () => {
    const grouped = groupSettingsByCategory(mockSettings);
    expect(Object.keys(grouped)).toHaveLength(3);
    expect(grouped["server"]).toHaveLength(2);
    expect(grouped["log"]).toHaveLength(1);
    expect(grouped["webui"]).toHaveLength(1);
  });

  it("returns empty object for empty array", () => {
    expect(groupSettingsByCategory([])).toEqual({});
  });

  it("preserves setting objects in groups", () => {
    const grouped = groupSettingsByCategory(mockSettings);
    expect(grouped["server"][0].key).toBe("server.port");
    expect(grouped["server"][1].key).toBe("server.host");
  });
});
