/**
 * Check if a string looks like a semantic version (e.g., "6.18.2", "6.12-rc1")
 */
function isSemanticVersion(val: string): boolean {
  return /^\d+\.\d+/.test(val);
}

/**
 * Compare two semantic version strings
 * Handles versions like "6.18.2", "6.2", "6.12-rc1", etc.
 */
function compareSemanticVersions(a: string, b: string): number {
  // Split into main version and suffix (e.g., "6.12-rc1" -> ["6.12", "rc1"])
  const [aMain, aSuffix] = a.split("-");
  const [bMain, bSuffix] = b.split("-");

  // Compare main version parts
  const aParts = aMain.split(".").map((p) => parseInt(p, 10) || 0);
  const bParts = bMain.split(".").map((p) => parseInt(p, 10) || 0);

  const maxLen = Math.max(aParts.length, bParts.length);
  for (let i = 0; i < maxLen; i++) {
    const aNum = aParts[i] || 0;
    const bNum = bParts[i] || 0;
    if (aNum !== bNum) {
      return aNum - bNum;
    }
  }

  // Main versions are equal, compare suffixes
  // No suffix (release) > suffix (pre-release like rc1)
  if (!aSuffix && bSuffix) return 1;
  if (aSuffix && !bSuffix) return -1;
  if (!aSuffix && !bSuffix) return 0;

  // Both have suffixes, compare them
  // Extract numeric part from suffix (e.g., "rc1" -> 1)
  const aRcMatch = aSuffix.match(/rc(\d+)/);
  const bRcMatch = bSuffix.match(/rc(\d+)/);

  if (aRcMatch && bRcMatch) {
    return parseInt(aRcMatch[1], 10) - parseInt(bRcMatch[1], 10);
  }

  // Fall back to string comparison for other suffixes
  return aSuffix.localeCompare(bSuffix);
}

/**
 * Smart comparison function that handles numeric, semantic version, and string values
 * @param aVal - First value to compare
 * @param bVal - Second value to compare
 * @param direction - Sort direction ('asc' or 'desc')
 * @returns Comparison result (-1, 0, or 1)
 */
export function compareValues(
  aVal: any,
  bVal: any,
  direction: "asc" | "desc" = "asc",
): number {
  const multiplier = direction === "asc" ? 1 : -1;

  // Try numeric comparison first
  const aNum = Number(aVal);
  const bNum = Number(bVal);

  if (!isNaN(aNum) && !isNaN(bNum)) {
    // Both are numbers
    if (aNum < bNum) return -1 * multiplier;
    if (aNum > bNum) return 1 * multiplier;
    return 0;
  }

  // Check for semantic version strings
  const aStr = String(aVal);
  const bStr = String(bVal);

  if (isSemanticVersion(aStr) && isSemanticVersion(bStr)) {
    return compareSemanticVersions(aStr, bStr) * multiplier;
  }

  // Fall back to string comparison
  const aLower = aStr.toLowerCase();
  const bLower = bStr.toLowerCase();

  if (aLower < bLower) return -1 * multiplier;
  if (aLower > bLower) return 1 * multiplier;
  return 0;
}
