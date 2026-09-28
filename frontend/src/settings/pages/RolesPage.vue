<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import Column from "primevue/column";
import DataTable from "primevue/datatable";
import { listRoles, type SettingsRole } from "../../auth/oidc";
import { useSettingsHelp } from "../help";
import SettingsPage from "../SettingsPage.vue";

defineProps<{ title: string }>();

const { t, locale } = useI18n();
const help = useSettingsHelp();
const roles = ref<SettingsRole[]>([]);
const selected = ref<SettingsRole | null>(null);
const loading = ref(false);
const error = ref("");

function description(role: SettingsRole): string {
  return locale.value === "de" ? role.description.de : role.description.en;
}

watch([selected, locale], () => {
  help.set(selected.value ? description(selected.value) : null);
});

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const items = await listRoles();
    roles.value = items;
    if (selected.value && !items.some((role) => role.name === selected.value?.name)) {
      selected.value = null;
    }
  } catch (e) {
    roles.value = [];
    selected.value = null;
    error.value = e instanceof Error && (e.message === "forbidden" || e.message === "not authenticated")
      ? t("settings.roles.forbidden")
      : t("settings.roles.loadError");
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>

<template>
  <SettingsPage :title="title">
    <p v-if="error" class="error">{{ error }}</p>
    <DataTable
      v-else
      v-model:selection="selected"
      :value="roles"
      data-key="name"
      selection-mode="single"
      :meta-key-selection="false"
      :loading="loading"
      size="small"
    >
      <template #empty>{{ t("settings.roles.empty") }}</template>
      <Column field="name" :header="t('settings.roles.name')" />
      <Column field="labels.de" :header="t('settings.roles.labelDe')">
        <template #body="{ data }">{{ data.labels.de }}</template>
      </Column>
      <Column field="labels.en" :header="t('settings.roles.labelEn')">
        <template #body="{ data }">{{ data.labels.en }}</template>
      </Column>
    </DataTable>
  </SettingsPage>
</template>

<style scoped>
.error {
  margin: 0;
  color: #9b2c2c;
}
</style>
