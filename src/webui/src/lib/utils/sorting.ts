/**
 * Smart comparison function that handles both numeric and string values
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

  // Fall back to string comparison
  const aStr = String(aVal).toLowerCase();
  const bStr = String(bVal).toLowerCase();

  if (aStr < bStr) return -1 * multiplier;
  if (aStr > bStr) return 1 * multiplier;
  return 0;
}
