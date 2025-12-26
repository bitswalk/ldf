// Locale types and utilities
export type SupportedLocale = "en" | "fr" | "de";
export type LocalePreference = "system" | SupportedLocale;

// Re-export from i18n service for convenience
export { i18nService, t } from "../services/i18n";
