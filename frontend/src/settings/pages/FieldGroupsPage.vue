<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import Button from "primevue/button";
import Column from "primevue/column";
import DataTable from "primevue/datatable";
import Dialog from "primevue/dialog";
import {
  createFieldGroup,
  deleteFieldGroup,
  fieldValueTypes,
  listFieldGroups,
  startAuthorization,
  updateFieldGroup,
  type FieldDefinition,
  type FieldGroup,
  type FieldGroupInput,
} from "../../auth/oidc";
import SettingsPage from "../SettingsPage.vue";

defineProps<{ title: string }>();

const { t, locale } = useI18n();
const groups = ref<FieldGroup[]>([]);
const selected = ref<FieldGroup | null>(null);
const loading = ref(false);
const error = ref("");

const editorOpen = ref(false);
const editingId = ref("");
const saving = ref(false);
const formError = ref("");
const form = ref<FieldGroupInput>(emptyForm());
const fieldOpen = ref<boolean[]>([]);

const deleteOpen = ref(false);
const deleting = ref(false);

const editorTitle = computed(() =>
  editingId.value ? t("settings.fieldGroups.editTitle") : t("settings.fieldGroups.createTitle"),
);

function emptyText() {
  return { de: "", en: "" };
}

function emptyForm(): FieldGroupInput {
  return { name: "", labels: emptyText(), description: emptyText(), fields: [] };
}

function emptyField(): FieldDefinition {
  return { name: "", labels: emptyText(), description: emptyText(), valueType: "text" };
}

function shownLabel(group: { name: string; labels: { de: string; en: string } }): string {
  const label = locale.value === "de" ? group.labels.de : group.labels.en;
  return label || group.name;
}

function actionMessage(key: string): string {
  switch (key) {
    case "already-exists":
      return t("settings.fieldGroups.alreadyExists");
    case "invalid-field-group":
      return t("settings.fieldGroups.invalid");
    case "forbidden":
      return t("settings.fieldGroups.forbidden");
    default:
      return t("settings.fieldGroups.actionError");
  }
}

function technicalNameOk(value: string): boolean {
  return /^[A-Za-z][A-Za-z0-9_]*$/.test(value.trim());
}

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const items = await listFieldGroups();
    groups.value = items;
    if (selected.value && !items.some((group) => group.id === selected.value?.id)) {
      selected.value = null;
    }
  } catch (e) {
    groups.value = [];
    selected.value = null;
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    error.value = e instanceof Error && e.message === "forbidden"
      ? t("settings.fieldGroups.forbidden")
      : t("settings.fieldGroups.loadError");
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editingId.value = "";
  form.value = emptyForm();
  fieldOpen.value = [];
  formError.value = "";
  editorOpen.value = true;
}

function openEdit() {
  const group = selected.value;
  if (!group) {
    return;
  }
  editingId.value = group.id;
  form.value = {
    name: group.name,
    labels: { ...group.labels },
    description: { ...group.description },
    fields: group.fields.map((field) => ({
      name: field.name,
      labels: { ...field.labels },
      description: { ...field.description },
      valueType: field.valueType,
    })),
  };
  fieldOpen.value = form.value.fields.map(() => false);
  formError.value = "";
  editorOpen.value = true;
}

function onRowDoubleClick(event: { data: FieldGroup }) {
  selected.value = event.data;
  openEdit();
}

function addField() {
  form.value.fields.push(emptyField());
  fieldOpen.value.push(true);
}

function removeField(index: number) {
  form.value.fields.splice(index, 1);
  fieldOpen.value.splice(index, 1);
}

function moveField(index: number, delta: number) {
  const next = index + delta;
  if (next < 0 || next >= form.value.fields.length) {
    return;
  }
  const fields = form.value.fields;
  const [item] = fields.splice(index, 1);
  fields.splice(next, 0, item);
  const [open] = fieldOpen.value.splice(index, 1);
  fieldOpen.value.splice(next, 0, open);
}

function toggleField(index: number) {
  fieldOpen.value[index] = !fieldOpen.value[index];
}

function fieldTitle(field: FieldDefinition, index: number): string {
  const name = field.name.trim();
  const label = shownLabel(field);
  if (name && label !== name) {
    return `${name} · ${label}`;
  }
  return name || label || String(index + 1);
}

function validate(input: FieldGroupInput): string {
  if (!technicalNameOk(input.name)) {
    return t("settings.fieldGroups.invalid");
  }
  const seen = new Set<string>();
  for (const field of input.fields) {
    if (!technicalNameOk(field.name) || !field.valueType) {
      return t("settings.fieldGroups.invalid");
    }
    const key = field.name.trim().toLowerCase();
    if (seen.has(key)) {
      return t("settings.fieldGroups.duplicateField");
    }
    seen.add(key);
  }
  return "";
}

function payload(): FieldGroupInput {
  return {
    name: form.value.name.trim(),
    labels: { de: form.value.labels.de.trim(), en: form.value.labels.en.trim() },
    description: { de: form.value.description.de.trim(), en: form.value.description.en.trim() },
    fields: form.value.fields.map((field) => ({
      name: field.name.trim(),
      labels: { de: field.labels.de.trim(), en: field.labels.en.trim() },
      description: { de: field.description.de.trim(), en: field.description.en.trim() },
      valueType: field.valueType,
    })),
  };
}

async function submit() {
  formError.value = "";
  const input = payload();
  const problem = validate(input);
  if (problem) {
    formError.value = problem;
    return;
  }
  saving.value = true;
  try {
    if (editingId.value) {
      await updateFieldGroup(editingId.value, input);
    } else {
      await createFieldGroup(input);
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
  const group = selected.value;
  if (!group) {
    return;
  }
  deleting.value = true;
  error.value = "";
  try {
    await deleteFieldGroup(group.id);
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
          :aria-label="t('settings.fieldGroups.create')"
          v-tooltip.bottom="t('settings.fieldGroups.create')"
          @click="openCreate"
        />
        <span v-tooltip.bottom="selected ? t('settings.fieldGroups.edit') : t('settings.fieldGroups.editNeedSelection')">
          <Button
            type="button"
            icon="pi pi-pencil"
            text
            rounded
            :aria-label="t('settings.fieldGroups.edit')"
            :disabled="!selected"
            @click="openEdit"
          />
        </span>
        <span v-tooltip.bottom="selected ? t('settings.fieldGroups.delete') : t('settings.fieldGroups.deleteNeedSelection')">
          <Button
            type="button"
            icon="pi pi-trash"
            text
            rounded
            severity="danger"
            :aria-label="t('settings.fieldGroups.delete')"
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
      :value="groups"
      data-key="id"
      selection-mode="single"
      :meta-key-selection="false"
      :loading="loading"
      size="small"
      @row-dblclick="onRowDoubleClick"
    >
      <template #empty>{{ t("settings.fieldGroups.empty") }}</template>
      <Column field="name" :header="t('settings.fieldGroups.name')" />
      <Column :header="t('settings.fieldGroups.label')">
        <template #body="{ data }">{{ shownLabel(data) }}</template>
      </Column>
      <Column :header="t('settings.fieldGroups.fieldCount')">
        <template #body="{ data }">{{ data.fields.length }}</template>
      </Column>
    </DataTable>

    <Dialog v-model:visible="editorOpen" modal :header="editorTitle" :style="{ width: '40rem' }">
      <form class="form" @submit.prevent="submit">
        <label v-if="editingId">
          {{ t("settings.fieldGroups.id") }}
          <input :value="editingId" readonly tabindex="-1" class="readonly" />
        </label>
        <label>
          {{ t("settings.fieldGroups.name") }}
          <input v-model="form.name" required autocomplete="off" />
        </label>
        <label>
          {{ t("settings.fieldGroups.labelDe") }}
          <input v-model="form.labels.de" autocomplete="off" />
        </label>
        <label>
          {{ t("settings.fieldGroups.labelEn") }}
          <input v-model="form.labels.en" autocomplete="off" />
        </label>
        <label>
          {{ t("settings.fieldGroups.descriptionDe") }}
          <textarea v-model="form.description.de" rows="2" />
        </label>
        <label>
          {{ t("settings.fieldGroups.descriptionEn") }}
          <textarea v-model="form.description.en" rows="2" />
        </label>
        <fieldset class="fields">
          <legend>{{ t("settings.fieldGroups.fields") }}</legend>
          <article v-for="(field, index) in form.fields" :key="index" class="field">
            <div class="field-head">
              <button
                type="button"
                class="field-toggle"
                :aria-expanded="fieldOpen[index]"
                :aria-label="fieldOpen[index] ? t('settings.fieldGroups.collapseField') : t('settings.fieldGroups.expandField')"
                @click="toggleField(index)"
              >
                <i class="pi" :class="fieldOpen[index] ? 'pi-chevron-down' : 'pi-chevron-right'" aria-hidden="true" />
                <span>{{ fieldTitle(field, index) }}</span>
                <span class="field-type">{{ t(`settings.fieldGroups.valueTypes.${field.valueType}`) }}</span>
              </button>
              <div class="field-actions">
                <Button type="button" icon="pi pi-arrow-up" text rounded :aria-label="t('settings.fieldGroups.moveUp')" :disabled="index === 0" @click="moveField(index, -1)" />
                <Button type="button" icon="pi pi-arrow-down" text rounded :aria-label="t('settings.fieldGroups.moveDown')" :disabled="index === form.fields.length - 1" @click="moveField(index, 1)" />
                <Button type="button" icon="pi pi-times" text rounded severity="danger" :aria-label="t('settings.fieldGroups.removeField')" @click="removeField(index)" />
              </div>
            </div>
            <template v-if="fieldOpen[index]">
            <label>
              {{ t("settings.fieldGroups.name") }}
              <input v-model="field.name" required autocomplete="off" />
            </label>
            <label>
              {{ t("settings.fieldGroups.valueType") }}
              <select v-model="field.valueType">
                <option v-for="valueType in fieldValueTypes" :key="valueType" :value="valueType">
                  {{ t(`settings.fieldGroups.valueTypes.${valueType}`) }}
                </option>
              </select>
            </label>
            <label>
              {{ t("settings.fieldGroups.labelDe") }}
              <input v-model="field.labels.de" autocomplete="off" />
            </label>
            <label>
              {{ t("settings.fieldGroups.labelEn") }}
              <input v-model="field.labels.en" autocomplete="off" />
            </label>
            <label>
              {{ t("settings.fieldGroups.descriptionDe") }}
              <textarea v-model="field.description.de" rows="2" />
            </label>
            <label>
              {{ t("settings.fieldGroups.descriptionEn") }}
              <textarea v-model="field.description.en" rows="2" />
            </label>
            </template>
          </article>
          <Button type="button" :label="t('settings.fieldGroups.addField')" text @click="addField" />
        </fieldset>
        <p v-if="formError" class="error">{{ formError }}</p>
      </form>
      <template #footer>
        <Button type="button" :label="t('settings.fieldGroups.cancel')" text @click="editorOpen = false" />
        <Button
          type="button"
          :label="editingId ? t('settings.fieldGroups.editSubmit') : t('settings.fieldGroups.createSubmit')"
          :loading="saving"
          @click="submit"
        />
      </template>
    </Dialog>

    <Dialog v-model:visible="deleteOpen" modal :header="t('settings.fieldGroups.deleteTitle')" :style="{ width: '24rem' }">
      <p>{{ t("settings.fieldGroups.deleteConfirm") }}</p>
      <p v-if="selected"><strong>{{ shownLabel(selected) }}</strong></p>
      <template #footer>
        <Button type="button" :label="t('settings.fieldGroups.cancel')" text @click="deleteOpen = false" />
        <Button type="button" severity="danger" :label="t('settings.fieldGroups.delete')" :loading="deleting" @click="submitDelete" />
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
.actions {
  display: flex;
  align-items: center;
  gap: 0.15rem;
}
.error {
  margin: 0;
  color: #9b2c2c;
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
.fields {
  margin: 0;
  padding: 0;
  border: 0;
  display: grid;
  gap: 0.75rem;
}
.field {
  display: grid;
  gap: 0.55rem;
  padding: 0.75rem;
  border: 1px solid #e4e8ec;
  border-radius: 10px;
  background: #f8f9fb;
}
.field-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.35rem;
}
.field-toggle {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  min-width: 0;
  flex: 1;
  padding: 0.15rem 0;
  border: 0;
  background: transparent;
  color: #1b1f24;
  font: inherit;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.field-toggle i {
  flex: 0 0 auto;
  font-size: 0.75rem;
  color: #5b6570;
}
.field-toggle span:first-of-type {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.field-type {
  flex: 0 0 auto;
  color: #5b6570;
  font-size: 0.8rem;
  font-weight: 500;
}
.field-actions {
  display: flex;
  align-items: center;
  flex: 0 0 auto;
}
</style>
