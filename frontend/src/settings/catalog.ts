import type { Component } from "vue";
import UsersPage from "./pages/UsersPage.vue";
import RolesPage from "./pages/RolesPage.vue";
import StoresPage from "./pages/StoresPage.vue";
import DocumentTypesPage from "./pages/DocumentTypesPage.vue";
import ExternalSystemsPage from "./pages/ExternalSystemsPage.vue";

/** A settings group or leaf. Only leaves map to a page. */
export type SettingsItem = {
  id: string;
  labelKey: string;
  helpKey: string;
  page?: Component;
  children?: SettingsItem[];
};

export type NavNode = {
  id: string;
  label: string;
  item: SettingsItem;
  children?: NavNode[];
};

export const settingsCatalog: SettingsItem[] = [
  {
    id: "users",
    labelKey: "settings.nav.users",
    helpKey: "settings.help.users",
    page: UsersPage,
  },
  {
    id: "roles",
    labelKey: "settings.nav.roles",
    helpKey: "settings.help.roles",
    page: RolesPage,
  },
  {
    id: "stores",
    labelKey: "settings.nav.stores",
    helpKey: "settings.help.stores",
    page: StoresPage,
  },
  {
    id: "document-types",
    labelKey: "settings.nav.documentTypes",
    helpKey: "settings.help.documentTypes",
    page: DocumentTypesPage,
  },
  {
    id: "external-systems",
    labelKey: "settings.nav.externalSystems",
    helpKey: "settings.help.externalSystems",
    page: ExternalSystemsPage,
  },
];

export function firstLeafId(items: SettingsItem[] = settingsCatalog): string | undefined {
  for (const item of items) {
    if (item.page && !item.children?.length) {
      return item.id;
    }
    if (item.children?.length) {
      const nested = firstLeafId(item.children);
      if (nested) {
        return nested;
      }
    }
  }
  return undefined;
}

export function pathToItem(id: string, items: SettingsItem[] = settingsCatalog, acc: SettingsItem[] = []): SettingsItem[] | undefined {
  for (const item of items) {
    const next = [...acc, item];
    if (item.id === id) {
      return next;
    }
    if (item.children?.length) {
      const found = pathToItem(id, item.children, next);
      if (found) {
        return found;
      }
    }
  }
  return undefined;
}

export function findItem(id: string, items: SettingsItem[] = settingsCatalog): SettingsItem | undefined {
  const path = pathToItem(id, items);
  return path?.[path.length - 1];
}
