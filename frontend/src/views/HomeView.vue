<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { fetchMe, startAuthorization } from "../auth/oidc";

type StoreKind = "open" | "readonly" | "folder";

type StoreEntry = {
  id: string;
  label: string;
  kind: StoreKind;
};

const { t } = useI18n();
const filter = ref("");
const error = ref("");
const userActions = ref<{ id: string; label: string }[]>([]);
const openSections = ref({ actions: userActions.value.length > 0, stores: true, status: true });

const stores = computed<StoreEntry[]>(() => [
  { id: "default", label: t("client.stores.default"), kind: "open" },
  { id: "audit", label: t("client.stores.audit"), kind: "readonly" },
  { id: "inbox", label: t("client.stores.inbox"), kind: "folder" },
]);

const statusRows = computed(() => [
  { id: "storage-default", label: t("client.status.storage", { name: t("client.stores.default") }), value: t("client.status.unknown") },
  { id: "docs-default", label: t("client.status.documents", { name: t("client.stores.default") }), value: t("client.status.none") },
  { id: "storage-audit", label: t("client.status.storage", { name: t("client.stores.audit") }), value: t("client.status.unknown") },
  { id: "docs-audit", label: t("client.status.documents", { name: t("client.stores.audit") }), value: t("client.status.none") },
  { id: "total", label: t("client.status.total"), value: t("client.status.unknown") },
  { id: "processing", label: t("client.status.processing"), value: t("client.status.none") },
  { id: "system", label: t("client.status.system"), value: t("client.status.ready") },
]);

const needle = computed(() => filter.value.trim().toLowerCase());

const visibleStores = computed(() => filterEntries(stores.value, needle.value));
const visibleStatus = computed(() => filterEntries(statusRows.value, needle.value));
const showActions = computed(() => matches(t("client.sections.actions"), needle.value) || matches(t("client.actionsEmpty"), needle.value));
const showStores = computed(() => matches(t("client.sections.stores"), needle.value) || visibleStores.value.length > 0);
const showStatus = computed(() => matches(t("client.sections.status"), needle.value) || visibleStatus.value.length > 0);

const storesToShow = computed(() => (matches(t("client.sections.stores"), needle.value) ? stores.value : visibleStores.value));
const statusToShow = computed(() => (matches(t("client.sections.status"), needle.value) ? statusRows.value : visibleStatus.value));

function matches(label: string, query: string) {
  return !query || label.toLowerCase().includes(query);
}

function filterEntries<T extends { label: string }>(items: T[], query: string) {
  if (!query) {
    return items;
  }
  return items.filter((item) => item.label.toLowerCase().includes(query));
}

function toggle(section: "actions" | "stores" | "status") {
  openSections.value[section] = !openSections.value[section];
}

function kindIcon(kind: StoreKind) {
  if (kind === "open") {
    return "pi pi-folder-open";
  }
  if (kind === "readonly") {
    return "pi pi-lock";
  }
  return "pi pi-inbox";
}

function kindLabel(kind: StoreKind) {
  if (kind === "open") {
    return t("client.stores.open");
  }
  if (kind === "readonly") {
    return t("client.stores.readonly");
  }
  return t("client.stores.folder");
}

onMounted(async () => {
  try {
    await fetchMe();
  } catch (e) {
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    error.value = e instanceof Error ? e.message : t("home.profileUnavailable");
  }
});
</script>

<template>
  <div class="dashboard">
    <nav class="nav" :aria-label="t('client.navLabel')">
      <input
        v-model="filter"
        type="search"
        class="filter"
        :placeholder="t('client.filterPlaceholder')"
        :aria-label="t('client.filterPlaceholder')"
      />
      <section v-if="showActions" class="block">
        <button type="button" class="head" :aria-expanded="openSections.actions" @click="toggle('actions')">
          <i class="pi" :class="openSections.actions ? 'pi-chevron-down' : 'pi-chevron-right'" aria-hidden="true" />
          {{ t("client.sections.actions") }}
        </button>
        <p v-if="openSections.actions && userActions.length === 0" class="hint">{{ t("client.actionsEmpty") }}</p>
        <ul v-else-if="openSections.actions" class="items">
          <li v-for="action in userActions" :key="action.id">{{ action.label }}</li>
        </ul>
      </section>
      <section v-if="showStores" class="block">
        <button type="button" class="head" :aria-expanded="openSections.stores" @click="toggle('stores')">
          <i class="pi" :class="openSections.stores ? 'pi-chevron-down' : 'pi-chevron-right'" aria-hidden="true" />
          {{ t("client.sections.stores") }}
        </button>
        <ul v-if="openSections.stores" class="items">
          <li v-for="store in storesToShow" :key="store.id">
            <span class="item">
              <i class="pi" :class="kindIcon(store.kind)" role="img" :title="kindLabel(store.kind)" :aria-label="kindLabel(store.kind)" />
              <span>{{ store.label }}</span>
            </span>
          </li>
        </ul>
      </section>
      <section v-if="showStatus" class="block">
        <button type="button" class="head" :aria-expanded="openSections.status" @click="toggle('status')">
          <i class="pi" :class="openSections.status ? 'pi-chevron-down' : 'pi-chevron-right'" aria-hidden="true" />
          {{ t("client.sections.status") }}
        </button>
        <dl v-if="openSections.status" class="status">
          <div v-for="row in statusToShow" :key="row.id">
            <dt>{{ row.label }}</dt>
            <dd>{{ row.value }}</dd>
          </div>
        </dl>
      </section>
    </nav>
    <section class="main">
      <p class="intro">{{ t("client.intro") }}</p>
      <p v-if="error" class="error">{{ error }}</p>
    </section>
  </div>
</template>

<style scoped>
.dashboard {
  display: grid;
  grid-template-columns: 17.5rem minmax(0, 1fr);
  flex: 1;
  min-height: 0;
  background: #fff;
}
.nav,
.main {
  min-height: 0;
  overflow: auto;
}
.nav {
  padding: 0.75rem 0.7rem 1rem;
  background: #f8f9fb;
  border-right: 1px solid #e4e8ec;
}
.filter {
  width: 100%;
  box-sizing: border-box;
  margin-bottom: 0.35rem;
  padding: 0.45rem 0.65rem;
  border: 1px solid #c9d0d6;
  border-radius: 8px;
  background: #fff;
  font: inherit;
}
.filter:focus {
  outline: 2px solid #1f4b99;
  outline-offset: 1px;
}
.block {
  margin-top: 0.35rem;
  padding-top: 0.35rem;
  border-top: 1px solid #e4e8ec;
}
.head {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  width: 100%;
  margin: 0;
  padding: 0.4rem 0.2rem;
  border: 0;
  background: transparent;
  color: #1b1f24;
  font: inherit;
  font-weight: 650;
  text-align: left;
  cursor: pointer;
}
.head i {
  font-size: 0.7rem;
  color: #5b6570;
}
.hint {
  margin: 0.15rem 0 0.45rem 1.15rem;
  color: #5b6570;
  font-size: 0.9rem;
  line-height: 1.4;
}
.items {
  margin: 0.15rem 0 0.4rem;
  padding: 0;
  list-style: none;
}
.item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.32rem 0.2rem 0.32rem 1.15rem;
  color: #1b1f24;
}
.item i {
  color: #1f4b99;
}
.item i.pi-lock {
  color: #5b6570;
}
.status {
  display: grid;
  gap: 0.45rem;
  margin: 0.2rem 0 0.5rem 1.15rem;
}
.status div {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
}
dt {
  color: #5b6570;
  font-size: 0.85rem;
}
dd {
  margin: 0;
  font-size: 0.85rem;
  font-variant-numeric: tabular-nums;
}
.main {
  padding: 1.35rem 1.6rem;
}
.intro {
  max-width: 40rem;
  margin: 0;
  font-size: 1.05rem;
  line-height: 1.5;
}
.error {
  color: #b42318;
}
</style>
