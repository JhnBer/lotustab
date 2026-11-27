import type { LocaleMessage } from "@intlify/core-base";
import { createI18n, type VueMessageType } from "vue-i18n";

export const LOCALE_OPTIONS = [
  { id: "en", name: "English" },
  { id: "zh-CN", name: "简体中文" },
  { id: "ja", name: "日本語" },
] as const;

export type AppLocale = (typeof LOCALE_OPTIONS)[number]["id"];

export const LOCALES = LOCALE_OPTIONS.map((item) => item.id) as readonly AppLocale[];

export const LOCALE_STORAGE_KEY = "lotustab.locale";

export function isAppLocale(value: unknown): value is AppLocale {
  return typeof value === "string" && (LOCALES as readonly string[]).includes(value);
}

export function readStoredLocale(): AppLocale {
  try {
    const stored = localStorage.getItem(LOCALE_STORAGE_KEY);
    return isAppLocale(stored) ? stored : "en";
  } catch {
    return "en";
  }
}

type LocaleCatalog = { [key: string]: string | LocaleCatalog };

const catalogs = import.meta.glob<LocaleCatalog>("./locales/*.json", {
  eager: true,
  import: "default",
});

function catalogId(path: string): string {
  const slash = path.lastIndexOf("/");
  const name = slash === -1 ? path : path.slice(slash + 1);
  return name.endsWith(".json") ? name.slice(0, -".json".length) : name;
}

function localeMessages(): { [K in AppLocale]: LocaleMessage<VueMessageType> } {
  const fileIds = Object.keys(catalogs).map(catalogId);
  const optionIds = LOCALES;
  const files = new Set(fileIds);
  const options = new Set<string>(optionIds);
  const missingFiles = optionIds.filter((id) => !files.has(id));
  const extraFiles = fileIds.filter((id) => !options.has(id));
  if (missingFiles.length > 0 || extraFiles.length > 0) {
    throw new Error(
      `locale catalogs and LOCALE_OPTIONS differ: missing files ${missingFiles.join(", ") || "none"}; extra files ${extraFiles.join(", ") || "none"}`,
    );
  }

  const messages = {} as { [K in AppLocale]: LocaleMessage<VueMessageType> };
  for (const [path, catalog] of Object.entries(catalogs)) {
    const id = catalogId(path);
    if (!isAppLocale(id)) {
      throw new Error(`unexpected locale catalog ${path}`);
    }
    messages[id] = catalog as LocaleMessage<VueMessageType>;
  }
  return messages;
}

export const i18n = createI18n({
  legacy: false,
  locale: readStoredLocale(),
  fallbackLocale: "en",
  messages: localeMessages(),
  missingWarn: false,
  fallbackWarn: false,
});

export function translate(key: string, params?: Record<string, unknown>): string {
  return params ? i18n.global.t(key, params) : i18n.global.t(key);
}
