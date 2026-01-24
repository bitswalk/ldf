// Glob pattern matching utility for version filtering

/**
 * Converts a glob pattern to a RegExp.
 * Supports:
 * - * matches any sequence of characters (except /)
 * - ? matches exactly one character
 * - ! prefix for negation (handled separately)
 */
function globToRegex(pattern: string): RegExp {
  // Escape special regex characters except * and ?
  let regexStr = pattern
    .replace(/[.+^${}()|[\]\\]/g, "\\$&")
    .replace(/\*/g, ".*")
    .replace(/\?/g, ".");

  return new RegExp(`^${regexStr}$`, "i");
}

export interface FilterPattern {
  pattern: string;
  regex: RegExp;
  isExclusion: boolean;
}

export interface FilterResult {
  version: string;
  included: boolean;
  reason?: string;
}

/**
 * Parses a comma-separated filter string into filter patterns.
 * Patterns starting with ! are exclusion patterns.
 */
export function parseVersionFilter(filterStr: string): FilterPattern[] {
  if (!filterStr || !filterStr.trim()) {
    return [];
  }

  return filterStr
    .split(",")
    .map((p) => p.trim())
    .filter((p) => p.length > 0)
    .map((pattern) => {
      const isExclusion = pattern.startsWith("!");
      const cleanPattern = isExclusion ? pattern.slice(1) : pattern;
      return {
        pattern,
        regex: globToRegex(cleanPattern),
        isExclusion,
      };
    });
}

/**
 * Checks if a version matches the given filter patterns.
 * Returns true if the version should be included.
 *
 * Logic:
 * - If there are only exclusion patterns, include by default unless excluded
 * - If there are inclusion patterns, exclude by default unless included
 * - Exclusion patterns always take precedence
 */
export function matchesFilter(
  version: string,
  patterns: FilterPattern[],
): boolean {
  if (patterns.length === 0) {
    return true;
  }

  const hasInclusionPatterns = patterns.some((p) => !p.isExclusion);

  // Check exclusion patterns first - if any match, exclude
  for (const pattern of patterns) {
    if (pattern.isExclusion && pattern.regex.test(version)) {
      return false;
    }
  }

  // If there are no inclusion patterns, include by default
  if (!hasInclusionPatterns) {
    return true;
  }

  // Check inclusion patterns - must match at least one
  for (const pattern of patterns) {
    if (!pattern.isExclusion && pattern.regex.test(version)) {
      return true;
    }
  }

  return false;
}

/**
 * Filters versions and returns detailed results with reasons.
 */
export function filterVersionsWithReasons(
  versions: string[],
  filterStr: string,
): FilterResult[] {
  const patterns = parseVersionFilter(filterStr);

  if (patterns.length === 0) {
    return versions.map((v) => ({ version: v, included: true }));
  }

  const hasInclusionPatterns = patterns.some((p) => !p.isExclusion);

  return versions.map((version) => {
    // Check exclusion patterns first
    for (const pattern of patterns) {
      if (pattern.isExclusion && pattern.regex.test(version)) {
        return {
          version,
          included: false,
          reason: `excluded by ${pattern.pattern}`,
        };
      }
    }

    // If there are no inclusion patterns, include by default
    if (!hasInclusionPatterns) {
      return { version, included: true };
    }

    // Check inclusion patterns
    for (const pattern of patterns) {
      if (!pattern.isExclusion && pattern.regex.test(version)) {
        return {
          version,
          included: true,
          reason: `matched ${pattern.pattern}`,
        };
      }
    }

    return {
      version,
      included: false,
      reason: "no inclusion pattern matched",
    };
  });
}

/**
 * Simple filter function that returns only included versions.
 */
export function filterVersions(
  versions: string[],
  filterStr: string,
): string[] {
  const patterns = parseVersionFilter(filterStr);
  return versions.filter((v) => matchesFilter(v, patterns));
}
