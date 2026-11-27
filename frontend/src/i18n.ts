import { createI18n } from "vue-i18n";
import en from "@/locales/en.json";
import zhCN from "@/locales/zh-CN.json";

export const LOCALES = ["en", "zh-CN"] as const;
export type AppLocale = (typeof LOCALES)[number];

export const LOCALE_STORAGE_KEY = "lotustab.locale";

export function isAppLocale(value: unknown): value is AppLocale {
  return value === "en" || value === "zh-CN";
}

export function readStoredLocale(): AppLocale {
  try {
    const stored = localStorage.getItem(LOCALE_STORAGE_KEY);
    return isAppLocale(stored) ? stored : "en";
  } catch {
    return "en";
  }
}

export const i18n = createI18n({
  legacy: false,
  locale: readStoredLocale(),
  fallbackLocale: "en",
  messages: {
    en,
    "zh-CN": zhCN,
  },
  missingWarn: false,
  fallbackWarn: false,
});

export function translate(key: string, params?: Record<string, unknown>): string {
  return params ? i18n.global.t(key, params) : i18n.global.t(key);
}
