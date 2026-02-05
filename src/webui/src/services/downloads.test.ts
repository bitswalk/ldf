import { describe, it, expect } from "vitest";
import {
  formatBytes,
  isJobActive,
  canRetryJob,
  getStatusDisplayText,
  getStatusColor,
} from "./downloads";
import type { DownloadJobStatus } from "./downloads";

describe("formatBytes", () => {
  it("returns '0 B' for zero bytes", () => {
    expect(formatBytes(0)).toBe("0 B");
  });

  it("formats bytes correctly", () => {
    expect(formatBytes(1)).toBe("1 B");
    expect(formatBytes(512)).toBe("512 B");
    expect(formatBytes(1023)).toBe("1023 B");
  });

  it("formats kilobytes correctly", () => {
    expect(formatBytes(1024)).toBe("1 KB");
    expect(formatBytes(1536)).toBe("1.5 KB");
  });

  it("formats megabytes correctly", () => {
    expect(formatBytes(1048576)).toBe("1 MB");
    expect(formatBytes(1572864)).toBe("1.5 MB");
  });

  it("formats gigabytes correctly", () => {
    expect(formatBytes(1073741824)).toBe("1 GB");
  });

  it("formats terabytes correctly", () => {
    expect(formatBytes(1099511627776)).toBe("1 TB");
  });
});

describe("isJobActive", () => {
  it("returns true for active statuses", () => {
    expect(isJobActive("pending")).toBe(true);
    expect(isJobActive("verifying")).toBe(true);
    expect(isJobActive("downloading")).toBe(true);
  });

  it("returns false for inactive statuses", () => {
    expect(isJobActive("completed")).toBe(false);
    expect(isJobActive("failed")).toBe(false);
    expect(isJobActive("cancelled")).toBe(false);
  });
});

describe("canRetryJob", () => {
  it("returns true for retryable statuses", () => {
    expect(canRetryJob("failed")).toBe(true);
    expect(canRetryJob("cancelled")).toBe(true);
  });

  it("returns false for non-retryable statuses", () => {
    expect(canRetryJob("pending")).toBe(false);
    expect(canRetryJob("downloading")).toBe(false);
    expect(canRetryJob("completed")).toBe(false);
    expect(canRetryJob("verifying")).toBe(false);
  });
});

describe("getStatusDisplayText", () => {
  const cases: [DownloadJobStatus, string][] = [
    ["pending", "Pending"],
    ["verifying", "Verifying"],
    ["downloading", "Downloading"],
    ["completed", "Completed"],
    ["failed", "Failed"],
    ["cancelled", "Cancelled"],
  ];

  it.each(cases)("maps '%s' to '%s'", (status, expected) => {
    expect(getStatusDisplayText(status)).toBe(expected);
  });
});

describe("getStatusColor", () => {
  it("returns correct color for each status", () => {
    expect(getStatusColor("pending")).toBe("default");
    expect(getStatusColor("downloading")).toBe("primary");
    expect(getStatusColor("verifying")).toBe("primary");
    expect(getStatusColor("completed")).toBe("success");
    expect(getStatusColor("failed")).toBe("danger");
    expect(getStatusColor("cancelled")).toBe("warning");
  });
});
