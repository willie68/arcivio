import { createI18n } from "vue-i18n";
import de from "./de";
import en from "./en";

export type AppLocale = "de" | "en";

export function detectLocale(): AppLocale {
  const languages = navigator.languages?.length ? navigator.languages : [navigator.language];
  return languages.some((lang) => lang.toLowerCase().startsWith("de")) ? "de" : "en";
}

export const i18n = createI18n({
  legacy: false,
  locale: detectLocale(),
  fallbackLocale: "en",
  messages: { de, en },
});

export function syncDocumentLang(locale: string) {
  document.documentElement.lang = locale;
}
