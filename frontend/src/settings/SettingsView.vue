<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { settingsCatalog, findItem, firstLeafId, pathToItem, type NavNode, type SettingsItem } from "./catalog";
import { provideSettingsHelp } from "./help";
import SettingsTree from "./SettingsTree.vue";

const { t } = useI18n();
const selectedId = ref(firstLeafId() ?? "");
const navOpen = ref(true);
const helpOpen = ref(true);
const helpOverride = provideSettingsHelp();

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
    icon: item.icon,
    item,
    children: item.children?.length ? toNavNodes(item.children) : undefined,
  }));
}

watch(selectedId, () => {
  helpOverride.value = null;
});

function onSelect(node: NavNode) {
  if (node.item.page) {
    selectedId.value = node.id;
  }
}
</script>

<template>
  <div class="settings" :class="{ 'nav-closed': !navOpen, 'help-closed': !helpOpen }">
    <nav class="pane nav" :aria-label="t('settings.navLabel')">
      <button
        type="button"
        class="nav-toggle"
        :aria-expanded="navOpen"
        :aria-label="navOpen ? t('settings.navCollapse') : t('settings.navExpand')"
        v-tooltip.right="navOpen ? undefined : t('settings.navExpand')"
        @click="navOpen = !navOpen"
      >
        <i class="pi" :class="navOpen ? 'pi-angle-left' : 'pi-angle-right'" aria-hidden="true" />
        <span v-if="navOpen">{{ t("settings.navLabel") }}</span>
      </button>
      <SettingsTree :nodes="nodes" :selected-id="selectedId" :collapsed="!navOpen" @select="onSelect" />
    </nav>
    <section class="pane main" aria-live="polite">
      <component :is="selectedItem?.page" v-if="selectedItem?.page" :title="pageTitle" />
    </section>
    <aside class="pane help" :aria-label="t('settings.helpTitle')">
      <button type="button" class="help-toggle" :aria-expanded="helpOpen" :aria-label="t('settings.helpTitle')" @click="helpOpen = !helpOpen">
        <template v-if="helpOpen">
          <i class="pi pi-angle-right" aria-hidden="true" />
          <span>{{ t("settings.helpTitle") }}</span>
        </template>
        <span v-else class="help-mark">?</span>
      </button>
      <p v-if="helpOpen && helpOverride">{{ helpOverride }}</p>
      <p v-else-if="helpOpen && selectedItem">{{ t(selectedItem.helpKey) }}</p>
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
.settings.nav-closed {
  grid-template-columns: 3.25rem minmax(0, 1fr) 17rem;
}
.settings.help-closed {
  grid-template-columns: 16.5rem minmax(0, 1fr) 2.4rem;
}
.settings.nav-closed.help-closed {
  grid-template-columns: 3.25rem minmax(0, 1fr) 2.4rem;
}
.pane {
  min-height: 0;
  overflow: auto;
}
.nav {
  display: flex;
  flex-direction: column;
  gap: 0.55rem;
  padding: 0.75rem 0.55rem 0.9rem;
  background: #f8f9fb;
  border-right: 1px solid #e4e8ec;
}
.nav-closed .nav {
  padding: 0.75rem 0.3rem 0.9rem;
  align-items: stretch;
}
.nav-toggle {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  width: 100%;
  margin: 0;
  padding: 0.15rem 0.2rem;
  border: 0;
  background: transparent;
  color: #5b6570;
  font: inherit;
  font-size: 0.85rem;
  font-weight: 650;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  text-align: left;
  cursor: pointer;
}
.nav-closed .nav-toggle {
  justify-content: center;
  padding: 0.35rem 0;
  text-transform: none;
  letter-spacing: 0;
}
.nav-toggle i {
  font-size: 0.85rem;
}
.main {
  padding: 1.25rem 1.5rem;
}
.help {
  padding: 0.85rem 1rem 1.15rem;
  background: #f8f9fb;
  border-left: 1px solid #e4e8ec;
  color: #5b6570;
}
.help-closed .help {
  padding: 0.75rem 0.3rem;
}
.help-toggle {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  width: 100%;
  margin: 0 0 0.7rem;
  padding: 0.15rem 0;
  border: 0;
  background: transparent;
  color: #5b6570;
  font: inherit;
  font-size: 0.85rem;
  font-weight: 650;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  text-align: left;
  cursor: pointer;
}
.help-closed .help-toggle {
  justify-content: center;
  margin: 0;
  text-transform: none;
  letter-spacing: 0;
}
.help-mark {
  font-size: 1.05rem;
  font-weight: 650;
  line-height: 1;
}
.help-toggle i {
  font-size: 0.85rem;
}
.help p {
  margin: 0;
  line-height: 1.45;
  font-size: 0.92rem;
}
</style>
