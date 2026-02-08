import { createSignal, createRoot, createMemo } from "solid-js";
import { flatten, translator, resolveTemplate } from "@solid-primitives/i18n";

// Types
export type SupportedLocale = "en" | "fr" | "de";
export type LocalePreference = "system" | SupportedLocale;

export interface Dictionary {
  [key: string]: string | Dictionary;
}

export interface FlatDictionary {
  [key: string]: string;
}

export interface LanguagePack {
  locale: string;
  name: string;
  dictionary: Dictionary;
  isCustom?: boolean;
}

// Storage keys
const STORAGE_KEYS = {
  LANGUAGE_PREFERENCE: "language-preference",
} as const;

// Supported locales configuration
const SUPPORTED_LOCALES: Array<{
  code: SupportedLocale;
  name: string;
  nativeName: string;
}> = [
  { code: "en", name: "English", nativeName: "English" },
  { code: "fr", name: "French", nativeName: "Francais" },
  { code: "de", name: "German", nativeName: "Deutsch" },
];

class I18nService {
  private locale!: ReturnType<typeof createSignal<SupportedLocale>>;
  private preference!: ReturnType<typeof createSignal<LocalePreference>>;
  private dictionary!: ReturnType<typeof createSignal<FlatDictionary>>;
  private isLoading!: ReturnType<typeof createSignal<boolean>>;
  private customLanguages!: ReturnType<
    typeof createSignal<Map<string, LanguagePack>>
  >;
  private disposeRoot?: () => void;

  // Bundled dictionaries cache
  private loadedDictionaries: Map<SupportedLocale, FlatDictionary> = new Map();

  constructor() {
    this.disposeRoot = createRoot((dispose) => {
      this.locale = createSignal<SupportedLocale>("en");
      this.preference = createSignal<LocalePreference>("system");
      this.dictionary = createSignal<FlatDictionary>({});
      this.isLoading = createSignal<boolean>(true);
      this.customLanguages = createSignal<Map<string, LanguagePack>>(new Map());
      return dispose;
    });
  }

  // Getters
  getLocale = () => this.locale[0]();
  getPreference = () => this.preference[0]();
  getDictionary = () => this.dictionary[0]();
  isLoadingLocale = () => this.isLoading[0]();
  getCustomLanguages = () => this.customLanguages[0]();

  // Translator function (memoized in components via createMemo)
  private createTranslator = () => {
    const dict = this.dictionary[0]();
    if (!dict || Object.keys(dict).length === 0) {
      return (key: string) => key;
    }
    return translator(() => dict, resolveTemplate);
  };

  // Main translation function
  t = (key: string, params?: Record<string, string | number>): string => {
    const translate = this.createTranslator();
    const result = translate(key, params);
    return typeof result === "string" ? result : key;
  };

  // Determine effective locale from system preference
  private getSystemLocale(): SupportedLocale {
    const browserLang = navigator.language.split("-")[0];
    const supported: SupportedLocale[] = ["en", "fr", "de"];
    return supported.includes(browserLang as SupportedLocale)
      ? (browserLang as SupportedLocale)
      : "en"; // Fallback to EN
  }

  // Load bundled locale files via dynamic import
  private async loadBundledLocale(
    locale: SupportedLocale,
  ): Promise<FlatDictionary> {
    // Check cache first
    const cached = this.loadedDictionaries.get(locale);
    if (cached) {
      return cached;
    }

    try {
      // Dynamic imports for code splitting
      const [
        common,
        settings,
        distribution,
        sources,
        artifacts,
        auth,
        components,
        build,
        boardProfiles,
        toolchainProfiles,
      ] = await Promise.all([
        import(`../locales/${locale}/common.json`),
        import(`../locales/${locale}/settings.json`),
        import(`../locales/${locale}/distribution.json`),
        import(`../locales/${locale}/sources.json`),
        import(`../locales/${locale}/artifacts.json`),
        import(`../locales/${locale}/auth.json`),
        import(`../locales/${locale}/components.json`),
        import(`../locales/${locale}/build.json`),
        import(`../locales/${locale}/boardProfiles.json`),
        import(`../locales/${locale}/toolchainProfiles.json`),
      ]);

      // Merge all modules with namespace prefixes
      const mergedDict: Dictionary = {
        common: common.default,
        settings: settings.default,
        distribution: distribution.default,
        sources: sources.default,
        artifacts: artifacts.default,
        auth: auth.default,
        components: components.default,
        build: build.default,
        boardProfiles: boardProfiles.default,
        toolchainProfiles: toolchainProfiles.default,
      };

      // Flatten for fast lookups
      const flatDict = flatten(mergedDict) as FlatDictionary;

      // Cache it
      this.loadedDictionaries.set(locale, flatDict);

      // Also cache to IndexedDB for offline support
      this.cacheDictionaryToIndexedDB(locale, flatDict);

      return flatDict;
    } catch (error) {
      console.error(`Failed to load locale ${locale}:`, error);

      // Try to load from IndexedDB cache
      const cachedDict = await this.getCachedDictionaryFromIndexedDB(locale);
      if (cachedDict) {
        return cachedDict;
      }

      // Fallback to English if not already English
      if (locale !== "en") {
        return this.loadBundledLocale("en");
      }

      // Return empty dict as last resort
      return {};
    }
  }

  // IndexedDB operations for offline support
  private async openCacheDB(): Promise<IDBDatabase> {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open("ldf-i18n", 1);
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve(request.result);
      request.onupgradeneeded = () => {
        const db = request.result;
        if (!db.objectStoreNames.contains("dictionaries")) {
          db.createObjectStore("dictionaries", { keyPath: "locale" });
        }
        if (!db.objectStoreNames.contains("custom")) {
          db.createObjectStore("custom", { keyPath: "locale" });
        }
      };
    });
  }

  private async getCachedDictionaryFromIndexedDB(
    locale: string,
  ): Promise<FlatDictionary | null> {
    try {
      const db = await this.openCacheDB();
      return new Promise((resolve) => {
        const tx = db.transaction("dictionaries", "readonly");
        const store = tx.objectStore("dictionaries");
        const request = store.get(locale);
        request.onsuccess = () => resolve(request.result?.dictionary || null);
        request.onerror = () => resolve(null);
      });
    } catch {
      return null;
    }
  }

  private async cacheDictionaryToIndexedDB(
    locale: string,
    dictionary: FlatDictionary,
  ): Promise<void> {
    try {
      const db = await this.openCacheDB();
      const tx = db.transaction("dictionaries", "readwrite");
      const store = tx.objectStore("dictionaries");
      store.put({ locale, dictionary, timestamp: Date.now() });
    } catch {
      // Silently fail - caching is optional
    }
  }

  // Set locale preference (system or explicit)
  async setPreference(preference: LocalePreference): Promise<void> {
    this.preference[1](preference);
    localStorage.setItem(STORAGE_KEYS.LANGUAGE_PREFERENCE, preference);

    const effectiveLocale =
      preference === "system" ? this.getSystemLocale() : preference;

    await this.setLocale(effectiveLocale);
  }

  // Set active locale and load dictionary
  async setLocale(locale: SupportedLocale): Promise<void> {
    this.isLoading[1](true);

    try {
      const dict = await this.loadBundledLocale(locale);
      this.locale[1](locale);
      this.dictionary[1](dict);
      document.documentElement.lang = locale;
    } catch (error) {
      console.error("Failed to set locale:", locale, error);
      // Fallback to English
      if (locale !== "en") {
        await this.setLocale("en");
      }
    } finally {
      this.isLoading[1](false);
    }
  }

  // Initialize service
  async initialize(): Promise<void> {
    // Load saved preference from localStorage
    const savedPreference = localStorage.getItem(
      STORAGE_KEYS.LANGUAGE_PREFERENCE,
    ) as LocalePreference | null;

    const preference = savedPreference || "system";
    this.preference[1](preference);

    const effectiveLocale =
      preference === "system" ? this.getSystemLocale() : preference;

    await this.setLocale(effectiveLocale);

    // Load any custom languages from IndexedDB
    await this.loadCustomLanguages();
  }

  // Load custom language packs from IndexedDB
  async loadCustomLanguages(): Promise<void> {
    try {
      const db = await this.openCacheDB();
      const tx = db.transaction("custom", "readonly");
      const store = tx.objectStore("custom");
      const request = store.getAll();

      request.onsuccess = () => {
        const packs = request.result as LanguagePack[];
        const map = new Map<string, LanguagePack>();
        packs.forEach((pack) => map.set(pack.locale, pack));
        this.customLanguages[1](map);
      };
    } catch {
      // Silently fail
    }
  }

  // Get all available locales (bundled + custom)
  getAvailableLocales(): Array<{
    code: string;
    name: string;
    nativeName: string;
    isCustom: boolean;
  }> {
    const bundled = SUPPORTED_LOCALES.map((l) => ({
      ...l,
      isCustom: false,
    }));

    const custom = Array.from(this.customLanguages[0]().entries()).map(
      ([code, pack]) => ({
        code,
        name: pack.name,
        nativeName: pack.name,
        isCustom: true,
      }),
    );

    return [...bundled, ...custom];
  }

  // Check if a locale is supported
  isLocaleSupported(locale: string): boolean {
    return (
      SUPPORTED_LOCALES.some((l) => l.code === locale) ||
      this.customLanguages[0]().has(locale)
    );
  }

  // Fetch custom language packs from backend
  async fetchLanguagePacksFromServer(): Promise<void> {
    try {
      const token = localStorage.getItem("auth-token");
      if (!token) {
        return; // Not authenticated
      }

      const serverUrl = localStorage.getItem("server-url") || "";
      const response = await fetch(`${serverUrl}/v1/language-packs`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        return; // Silently fail
      }

      const data = await response.json();
      const packs = data.language_packs || [];

      // Update custom languages map
      const map = new Map<string, LanguagePack>();
      for (const pack of packs) {
        // Fetch full dictionary for each pack
        const fullPack = await this.fetchLanguagePackDictionary(pack.locale);
        if (fullPack) {
          map.set(pack.locale, {
            locale: pack.locale,
            name: pack.name,
            dictionary: fullPack.dictionary,
            isCustom: true,
          });
          // Cache to IndexedDB
          await this.cacheCustomLanguagePack(fullPack);
        }
      }
      this.customLanguages[1](map);
    } catch {
      // Silently fail - will use cached packs
    }
  }

  // Fetch a specific language pack dictionary from server
  private async fetchLanguagePackDictionary(
    locale: string,
  ): Promise<{ locale: string; name: string; dictionary: Dictionary } | null> {
    try {
      const token = localStorage.getItem("auth-token");
      const serverUrl = localStorage.getItem("server-url") || "";
      const response = await fetch(`${serverUrl}/v1/language-packs/${locale}`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        return null;
      }

      const data = await response.json();
      return {
        locale: data.locale,
        name: data.name,
        dictionary: data.dictionary,
      };
    } catch {
      return null;
    }
  }

  // Cache custom language pack to IndexedDB
  private async cacheCustomLanguagePack(pack: {
    locale: string;
    name: string;
    dictionary: Dictionary;
  }): Promise<void> {
    try {
      const db = await this.openCacheDB();
      const tx = db.transaction("custom", "readwrite");
      const store = tx.objectStore("custom");
      const flatDict = flatten(pack.dictionary) as FlatDictionary;
      store.put({
        locale: pack.locale,
        name: pack.name,
        dictionary: flatDict,
        timestamp: Date.now(),
      });
    } catch {
      // Silently fail
    }
  }

  // Install custom language pack via file upload
  async installLanguagePack(
    file: File,
  ): Promise<{ success: boolean; locale?: string; error?: string }> {
    try {
      const token = localStorage.getItem("auth-token");
      if (!token) {
        return { success: false, error: "Not authenticated" };
      }

      const serverUrl = localStorage.getItem("server-url") || "";
      const formData = new FormData();
      formData.append("file", file);

      const response = await fetch(`${serverUrl}/v1/language-packs`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
        },
        body: formData,
      });

      const data = await response.json();

      if (!response.ok) {
        return {
          success: false,
          error: data.message || "Failed to upload language pack",
        };
      }

      // Refresh custom languages list
      await this.fetchLanguagePacksFromServer();

      return {
        success: true,
        locale: data.locale,
      };
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : "Unknown error",
      };
    }
  }

  // Delete a custom language pack
  async deleteLanguagePack(
    locale: string,
  ): Promise<{ success: boolean; error?: string }> {
    try {
      const token = localStorage.getItem("auth-token");
      if (!token) {
        return { success: false, error: "Not authenticated" };
      }

      const serverUrl = localStorage.getItem("server-url") || "";
      const response = await fetch(`${serverUrl}/v1/language-packs/${locale}`, {
        method: "DELETE",
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      const data = await response.json();

      if (!response.ok) {
        return {
          success: false,
          error: data.message || "Failed to delete language pack",
        };
      }

      // Remove from local cache
      const map = new Map(this.customLanguages[0]());
      map.delete(locale);
      this.customLanguages[1](map);

      // Remove from IndexedDB
      try {
        const db = await this.openCacheDB();
        const tx = db.transaction("custom", "readwrite");
        const store = tx.objectStore("custom");
        store.delete(locale);
      } catch {
        // Silently fail
      }

      return { success: true };
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : "Unknown error",
      };
    }
  }

  // List language packs from server (for admin UI)
  async listLanguagePacksFromServer(): Promise<{
    success: boolean;
    packs?: Array<{
      locale: string;
      name: string;
      version: string;
      author?: string;
    }>;
    error?: string;
  }> {
    try {
      const token = localStorage.getItem("auth-token");
      if (!token) {
        return { success: false, error: "Not authenticated" };
      }

      const serverUrl = localStorage.getItem("server-url") || "";
      const response = await fetch(`${serverUrl}/v1/language-packs`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        const data = await response.json();
        return {
          success: false,
          error: data.message || "Failed to fetch language packs",
        };
      }

      const data = await response.json();
      return {
        success: true,
        packs: data.language_packs || [],
      };
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : "Unknown error",
      };
    }
  }
}

// Singleton export
export const i18nService = new I18nService();

// Convenience export for translation function
export const t = i18nService.t;
