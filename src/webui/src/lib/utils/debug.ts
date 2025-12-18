// Debug utilities - only log when dev-mode is enabled in localStorage

export function isDevMode(): boolean {
  return localStorage.getItem("dev-mode") === "true";
}

export function setDevModeLocal(enabled: boolean): void {
  if (enabled) {
    localStorage.setItem("dev-mode", "true");
  } else {
    localStorage.removeItem("dev-mode");
  }
  // Dispatch custom event to notify components in the same tab
  window.dispatchEvent(
    new CustomEvent("devmode-change", { detail: { enabled } }),
  );
}

export function debugLog(...args: unknown[]): void {
  if (isDevMode()) {
    console.log(...args);
  }
}

export function debugError(...args: unknown[]): void {
  if (isDevMode()) {
    console.error(...args);
  }
}

export function debugWarn(...args: unknown[]): void {
  if (isDevMode()) {
    console.warn(...args);
  }
}
