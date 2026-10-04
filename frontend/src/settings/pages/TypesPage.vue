<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import Button from "primevue/button";
import Column from "primevue/column";
import DataTable from "primevue/datatable";
import Dialog from "primevue/dialog";
import {
  createDocumentType,
  deleteDocumentType,
  listDocumentTypes,
  listFieldGroups,
  startAuthorization,
  updateDocumentType,
  type DocumentType,
  type DocumentTypeInput,
  type FieldGroup,
} from "../../auth/oidc";
import SettingsPage from "../SettingsPage.vue";

defineProps<{ title: string }>();

const { t, locale } = useI18n();
const types = ref<DocumentType[]>([]);
const groups = ref<FieldGroup[]>([]);
const selected = ref<DocumentType | null>(null);
const loading = ref(false);
const error = ref("");

const editorOpen = ref(false);
const editingId = ref("");
const saving = ref(false);
const formError = ref("");
const form = ref<DocumentTypeInput>(emptyForm());
const groupToAdd = ref("");

const deleteOpen = ref(false);
const deleting = ref(false);

const editorTitle = computed(() =>
  editingId.value ? t("settings.types.editTitle") : t("settings.types.createTitle"),
);
const availableGroups = computed(() => groups.value.filter((group) => !form.value.fieldGroupIds.includes(group.id)));

function emptyText() {
  return { de: "", en: "" };
}

function emptyForm(): DocumentTypeInput {
  return { name: "", labels: emptyText(), description: emptyText(), fieldGroupIds: [] };
}

function shownLabel(item: { name: string; labels: { de: string; en: string } }): string {
  const label = locale.value === "de" ? item.labels.de : item.labels.en;
  return label || item.name;
}

function groupLabel(id: string): string {
  const group = groups.value.find((item) => item.id === id);
  return group ? shownLabel(group) : t("settings.types.missingGroup");
}

function actionMessage(key: string): string {
  switch (key) {
    case "already-exists":
      return t("settings.types.alreadyExists");
    case "invalid-document-type":
      return t("settings.types.invalid");
    case "forbidden":
      return t("settings.types.forbidden");
    default:
      return t("settings.types.actionError");
  }
}

function technicalNameOk(value: string): boolean {
  return /^[A-Za-z][A-Za-z0-9_]*$/.test(value.trim());
}

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const [typeItems, groupItems] = await Promise.all([listDocumentTypes(), listFieldGroups()]);
    types.value = typeItems;
    groups.value = groupItems;
    if (selected.value && !typeItems.some((item) => item.id === selected.value?.id)) {
      selected.value = null;
    }
  } catch (e) {
    types.value = [];
    groups.value = [];
    selected.value = null;
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    error.value = e instanceof Error && e.message === "forbidden"
      ? t("settings.types.forbidden")
      : t("settings.types.loadError");
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editingId.value = "";
  form.value = emptyForm();
  groupToAdd.value = "";
  formError.value = "";
  editorOpen.value = true;
}

function openEdit() {
  const item = selected.value;
  if (!item) {
    return;
  }
  editingId.value = item.id;
  form.value = {
    name: item.name,
    labels: { ...item.labels },
    description: { ...item.description },
    fieldGroupIds: [...item.fieldGroupIds],
  };
  groupToAdd.value = "";
  formError.value = "";
  editorOpen.value = true;
}

function onRowDoubleClick(event: { data: DocumentType }) {
  selected.value = event.data;
  openEdit();
}

function addGroup() {
  const id = groupToAdd.value;
  if (!id || form.value.fieldGroupIds.includes(id)) {
    return;
  }
  form.value.fieldGroupIds.push(id);
  groupToAdd.value = "";
}

function removeGroup(index: number) {
  form.value.fieldGroupIds.splice(index, 1);
}

function moveGroup(index: number, delta: number) {
  const next = index + delta;
  const ids = form.value.fieldGroupIds;
  if (next < 0 || next >= ids.length) {
    return;
  }
  const [item] = ids.splice(index, 1);
  ids.splice(next, 0, item);
}

function payload(): DocumentTypeInput {
  return {
    name: form.value.name.trim(),
    labels: { de: form.value.labels.de.trim(), en: form.value.labels.en.trim() },
    description: { de: form.value.description.de.trim(), en: form.value.description.en.trim() },
    fieldGroupIds: [...form.value.fieldGroupIds],
  };
}

async function submit() {
  formError.value = "";
  const input = payload();
  if (!technicalNameOk(input.name)) {
    formError.value = t("settings.types.invalid");
    return;
  }
  saving.value = true;
  try {
    if (editingId.value) {
      await updateDocumentType(editingId.value, input);
    } else {
      await createDocumentType(input);
    }
    editorOpen.value = false;
    await load();
  } catch (e) {
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    formError.value = actionMessage(e instanceof Error ? e.message : "");
  } finally {
    saving.value = false;
  }
}

function openDelete() {
  if (!selected.value) {
    return;
  }
  deleteOpen.value = true;
}

async function submitDelete() {
  const item = selected.value;
  if (!item) {
    return;
  }
  deleting.value = true;
  error.value = "";
  try {
    await deleteDocumentType(item.id);
    selected.value = null;
    deleteOpen.value = false;
    await load();
  } catch (e) {
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    error.value = actionMessage(e instanceof Error ? e.message : "");
    deleteOpen.value = false;
  } finally {
    deleting.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>

<template>
  <SettingsPage :title="title">
    <div class="toolbar">
      <div class="actions">
        <Button
          type="button"
          icon="pi pi-plus"
          text
          rounded
          :aria-label="t('settings.types.create')"
          v-tooltip.bottom="t('settings.types.create')"
          @click="openCreate"
        />
        <span v-tooltip.bottom="selected ? t('settings.types.edit') : t('settings.types.editNeedSelection')">
          <Button
            type="button"
            icon="pi pi-pencil"
            text
            rounded
            :aria-label="t('settings.types.edit')"
            :disabled="!selected"
            @click="openEdit"
          />
        </span>
        <span v-tooltip.bottom="selected ? t('settings.types.delete') : t('settings.types.deleteNeedSelection')">
          <Button
            type="button"
            icon="pi pi-trash"
            text
            rounded
            severity="danger"
            :aria-label="t('settings.types.delete')"
            :disabled="!selected || deleting"
            @click="openDelete"
          />
        </span>
      </div>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
    <DataTable
      v-else
      v-model:selection="selected"
      :value="types"
      data-key="id"
      selection-mode="single"
      :meta-key-selection="false"
      :loading="loading"
      size="small"
      @row-dblclick="onRowDoubleClick"
    >
      <template #empty>{{ t("settings.types.empty") }}</template>
      <Column field="name" :header="t('settings.types.name')" />
      <Column :header="t('settings.types.label')">
        <template #body="{ data }">{{ shownLabel(data) }}</template>
      </Column>
      <Column :header="t('settings.types.fieldGroupCount')">
        <template #body="{ data }">{{ data.fieldGroupIds.length }}</template>
      </Column>
    </DataTable>

    <Dialog v-model:visible="editorOpen" modal :header="editorTitle" :style="{ width: '36rem' }">
      <form class="form" @submit.prevent="submit">
        <label v-if="editingId">
          {{ t("settings.types.id") }}
          <input :value="editingId" readonly tabindex="-1" class="readonly" />
        </label>
        <label>
          {{ t("settings.types.name") }}
          <input v-model="form.name" required autocomplete="off" />
        </label>
        <label>
          {{ t("settings.types.labelDe") }}
          <input v-model="form.labels.de" autocomplete="off" />
        </label>
        <label>
          {{ t("settings.types.labelEn") }}
          <input v-model="form.labels.en" autocomplete="off" />
        </label>
        <label>
          {{ t("settings.types.descriptionDe") }}
          <textarea v-model="form.description.de" rows="2" />
        </label>
        <label>
          {{ t("settings.types.descriptionEn") }}
          <textarea v-model="form.description.en" rows="2" />
        </label>
        <fieldset class="groups">
          <legend>{{ t("settings.types.fieldGroups") }}</legend>
          <p v-if="!groups.length" class="hint">{{ t("settings.types.noGroups") }}</p>
          <div v-else class="add-group">
            <select v-model="groupToAdd" :aria-label="t('settings.types.chooseGroup')">
              <option value="">{{ t("settings.types.chooseGroup") }}</option>
              <option v-for="group in availableGroups" :key="group.id" :value="group.id">{{ shownLabel(group) }}</option>
            </select>
            <Button type="button" :label="t('settings.types.addGroup')" text :disabled="!groupToAdd" @click="addGroup" />
          </div>
          <article v-for="(id, index) in form.fieldGroupIds" :key="id" class="group">
            <span>{{ groupLabel(id) }}</span>
            <div class="group-actions">
              <Button type="button" icon="pi pi-arrow-up" text rounded :aria-label="t('settings.types.moveUp')" :disabled="index === 0" @click="moveGroup(index, -1)" />
              <Button type="button" icon="pi pi-arrow-down" text rounded :aria-label="t('settings.types.moveDown')" :disabled="index === form.fieldGroupIds.length - 1" @click="moveGroup(index, 1)" />
              <Button type="button" icon="pi pi-times" text rounded severity="danger" :aria-label="t('settings.types.removeGroup')" @click="removeGroup(index)" />
            </div>
          </article>
        </fieldset>
        <p v-if="formError" class="error">{{ formError }}</p>
      </form>
      <template #footer>
        <Button type="button" :label="t('settings.types.cancel')" text @click="editorOpen = false" />
        <Button
          type="button"
          :label="editingId ? t('settings.types.editSubmit') : t('settings.types.createSubmit')"
          :loading="saving"
          @click="submit"
        />
      </template>
    </Dialog>

    <Dialog v-model:visible="deleteOpen" modal :header="t('settings.types.deleteTitle')" :style="{ width: '24rem' }">
      <p>{{ t("settings.types.deleteConfirm") }}</p>
      <p v-if="selected"><strong>{{ shownLabel(selected) }}</strong></p>
      <template #footer>
        <Button type="button" :label="t('settings.types.cancel')" text @click="deleteOpen = false" />
        <Button type="button" severity="danger" :label="t('settings.types.delete')" :loading="deleting" @click="submitDelete" />
      </template>
    </Dialog>
  </SettingsPage>
</template>

<style scoped>
.toolbar {
  display: flex;
  justify-content: flex-end;
  margin: 0 0 0.85rem;
}
.actions,
.group-actions,
.add-group {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}
.error {
  margin: 0;
  color: #9b2c2c;
}
.hint {
  margin: 0;
  color: #5b6570;
}
.form {
  display: grid;
  gap: 0.75rem;
}
.form label,
.form legend {
  display: grid;
  gap: 0.3rem;
  color: #243040;
  font-size: 0.9rem;
}
.form input,
.form textarea,
.form select {
  padding: 0.45rem 0.6rem;
  border: 1px solid #c9d0d6;
  border-radius: 8px;
  font: inherit;
}
.form input.readonly {
  background: #f3f5f7;
  color: #5a6672;
  font-family: ui-monospace, monospace;
  font-size: 0.9rem;
}
.add-group select {
  flex: 1;
}
.groups {
  margin: 0;
  padding: 0;
  border: 0;
  display: grid;
  gap: 0.55rem;
}
.group {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.35rem 0.65rem;
  border: 1px solid #e4e8ec;
  border-radius: 10px;
  background: #f8f9fb;
}
</style>
