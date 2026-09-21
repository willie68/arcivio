<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { settingsCatalog, findItem, firstLeafId, pathToItem, type NavNode, type SettingsItem } from "./catalog";
import SettingsTree from "./SettingsTree.vue";

const { t } = useI18n();
const selectedId = ref(firstLeafId() ?? "");

const nodes = computed<NavNode[]>(() => toNavNodes(settingsCatalog));
const selectedItem = computed(() => (selectedId.value ? findItem(selectedId.value) : undefined));
const pageTitle = computed(() => {
  const path = selectedId.value ? pathToItem(selectedId.value) : undefined;
  return (path ?? []).map((item) => t(item.labelKey)).join(" - ");
});

function toNavNodes(items: SettingsItem[]): NavNode[] {
  return items.map((item) => ({
    id: item.id,
    label: t(item.labelKey),
    item,
    children: item.children?.length ? toNavNodes(item.children) : undefined,
  }));
}

function onSelect(node: NavNode) {
  if (node.item.page) {
    selectedId.value = node.id;
  }
}
</script>

<template>
  <div class="settings">
    <nav class="pane nav" :aria-label="t('settings.navLabel')">
      <SettingsTree :nodes="nodes" :selected-id="selectedId" @select="onSelect" />
    </nav>
    <section class="pane main" aria-live="polite">
      <component :is="selectedItem?.page" v-if="selectedItem?.page" :title="pageTitle" />
    </section>
    <aside class="pane help" :aria-label="t('settings.helpTitle')">
      <h2>{{ t("settings.helpTitle") }}</h2>
      <p v-if="selectedItem">{{ t(selectedItem.helpKey) }}</p>
    </aside>
  </div>
</template>

<style scoped>
.settings {
  display: grid;
  grid-template-columns: 16.5rem minmax(0, 1fr) 17rem;
  flex: 1;
  min-height: 0;
  background: #fff;
}
.pane {
  min-height: 0;
  overflow: auto;
}
.nav {
  padding: 0.75rem 0.55rem 0.9rem;
  background: #f8f9fb;
  border-right: 1px solid #e4e8ec;
}
.main {
  padding: 1.25rem 1.5rem;
}
.help {
  padding: 1.15rem 1.15rem 1.35rem;
  background: #f8f9fb;
  border-left: 1px solid #e4e8ec;
  color: #5b6570;
}
.help h2 {
  margin: 0 0 0.7rem;
  font-size: 0.85rem;
  font-weight: 650;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: #5b6570;
}
.help p {
  margin: 0;
  line-height: 1.45;
  font-size: 0.92rem;
}
</style>
