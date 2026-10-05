<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import Button from "primevue/button";
import Column from "primevue/column";
import DataTable from "primevue/datatable";
import Dialog from "primevue/dialog";
import Menu from "primevue/menu";
import {
  createDocumentType,
  deleteDocumentType,
  exportDocumentTypes,
  importDocumentTypes,
  previewDocumentTypeImport,
  listDocumentTypes,
  listFieldGroups,
  startAuthorization,
  systemFieldGroupId,
  updateDocumentType,
  type DocumentType,
  type DocumentTypeExchange,
  type DocumentTypeInput,
  type ImportConflict,
  type ImportDecision,
  type FieldGroup,
} from "../../auth/oidc";
import SettingsPage from "../SettingsPage.vue";

defineProps<{ title: string }>();

const { t, locale } = useI18n();
const types = ref<DocumentType[]>([]);
const catalog = ref<FieldGroup[]>([]);
const selected = ref<DocumentType | null>(null);
const loading = ref(false);
const error = ref("");

const editorOpen = ref(false);
const editingId = ref("");
const saving = ref(false);
const formError = ref("");
const form = ref<DocumentTypeInput>(emptyForm());
const addGroupId = ref("");

const deleteOpen = ref(false);
const deleting = ref(false);
const moreMenu = ref<{ toggle: (event: Event) => void } | null>(null);
const importInput = ref<HTMLInputElement | null>(null);
const transferring = ref(false);
const notice = ref("");
const noticeIsError = ref(false);
const pendingExchange = ref<DocumentTypeExchange | null>(null);
const conflicts = ref<ImportConflict[]>([]);
const conflictIndex = ref(0);
const decisions = ref<ImportDecision[]>([]);
const conflictOpen = ref(false);
const conflictSettled = ref(false);
const renameName = ref("");
const conflictError = ref("");

const editorTitle = computed(() =>
  editingId.value ? t("settings.types.editTitle") : t("settings.types.createTitle"),
);
const editTip = computed(() =>
  selected.value ? t("settings.types.edit") : t("settings.types.editNeedSelection"),
);
const deleteTip = computed(() =>
  selected.value ? t("settings.types.delete") : t("settings.types.deleteNeedSelection"),
);
const moreItems = computed(() => [
  { label: t("settings.types.export"), icon: "pi pi-download", command: () => void exportCatalog() },
  { label: t("settings.types.import"), icon: "pi pi-upload", command: () => importInput.value?.click() },
]);
const availableGroups = computed(() =>
  catalog.value.filter((group) => !form.value.fieldGroups.includes(group.id)),
);

function emptyText() {
  return { de: "", en: "" };
}

function emptyForm(): DocumentTypeInput {
  return {
    name: "",
    labels: emptyText(),
    description: emptyText(),
    fieldGroups: [systemFieldGroupId],
  };
}

function shownLabel(item: { name: string; labels: { de: string; en: string } }): string {
  const label = locale.value === "de" ? item.labels.de : item.labels.en;
  return label || item.name;
}

function groupById(id: string): FieldGroup | undefined {
  return catalog.value.find((group) => group.id === id);
}

function groupLabel(id: string): string {
  const group = groupById(id);
  return group ? shownLabel(group) : id;
}

function isSystem(id: string): boolean {
  return id === systemFieldGroupId;
}

function actionMessage(key: string): string {
  switch (key) {
    case "already-exists":
      return t("settings.types.alreadyExists");
    case "system-group":
      return t("settings.types.systemGroup");
    case "unknown-field-group":
      return t("settings.types.unknownGroup");
    case "invalid-field-group":
      return t("settings.types.invalidGroup");
    case "invalid-document-type":
      return t("settings.types.invalid");
    case "import-conflict":
      return t("settings.types.importConflict");
    case "forbidden":
      return t("settings.types.forbidden");
    default:
      return t("settings.types.actionError");
  }
}

function technicalNameOk(value: string): boolean {
  return /^[A-Za-z][A-Za-z0-9_ ()]*$/.test(value.trim());
}

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const [items, groups] = await Promise.all([listDocumentTypes(), listFieldGroups()]);
    types.value = items;
    catalog.value = groups;
    if (selected.value && !items.some((docType) => docType.id === selected.value?.id)) {
      selected.value = null;
    }
  } catch (e) {
    types.value = [];
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
  addGroupId.value = "";
  formError.value = "";
  editorOpen.value = true;
}

function copyType(docType: DocumentType): DocumentTypeInput {
  const groups = [...docType.fieldGroups];
  if (!groups.includes(systemFieldGroupId)) {
    groups.unshift(systemFieldGroupId);
  }
  return {
    name: docType.name,
    labels: { ...docType.labels },
    description: { ...docType.description },
    fieldGroups: groups,
  };
}

function openEdit() {
  const docType = selected.value;
  if (!docType) {
    return;
  }
  editingId.value = docType.id;
  form.value = copyType(docType);
  addGroupId.value = "";
  formError.value = "";
  editorOpen.value = true;
}

function onRowDoubleClick(event: { data: DocumentType }) {
  selected.value = event.data;
  openEdit();
}

function addGroup() {
  const id = addGroupId.value;
  if (!id || form.value.fieldGroups.includes(id)) {
    return;
  }
  form.value.fieldGroups.push(id);
  addGroupId.value = "";
}

function removeGroup(index: number) {
  if (isSystem(form.value.fieldGroups[index])) {
    return;
  }
  form.value.fieldGroups.splice(index, 1);
}

function moveGroup(index: number, delta: number) {
  const next = index + delta;
  if (next < 0 || next >= form.value.fieldGroups.length) {
    return;
  }
  const groups = form.value.fieldGroups;
  const current = groups[index];
  groups[index] = groups[next];
  groups[next] = current;
}

function validate(input: DocumentTypeInput): string {
  if (!technicalNameOk(input.name) || !input.fieldGroups.includes(systemFieldGroupId)) {
    return !input.fieldGroups.includes(systemFieldGroupId)
      ? t("settings.types.systemGroup")
      : t("settings.types.invalid");
  }
  return "";
}

function payload(): DocumentTypeInput {
  return {
    name: form.value.name.trim(),
    labels: { de: form.value.labels.de.trim(), en: form.value.labels.en.trim() },
    description: { de: form.value.description.de.trim(), en: form.value.description.en.trim() },
    fieldGroups: [...form.value.fieldGroups],
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

function openMore(event: Event) {
  moreMenu.value?.toggle(event);
}

function downloadJson(data: unknown, filename: string) {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

function isExchange(value: unknown): value is DocumentTypeExchange {
  if (!value || typeof value !== "object") {
    return false;
  }
  const body = value as DocumentTypeExchange;
  const typesOk = Array.isArray(body.documentTypes) && body.documentTypes.every((item) => hasId(item) && Array.isArray(item.fieldGroups));
  const groupsOk = body.fieldGroups == null || (Array.isArray(body.fieldGroups) && body.fieldGroups.every((item) => hasId(item)));
  return typesOk && groupsOk;
}

function hasId(item: { id?: string }): boolean {
  return typeof item?.id === "string" && item.id.trim() !== "";
}

const currentConflict = computed(() => conflicts.value[conflictIndex.value] ?? null);
const conflictTitle = computed(() =>
  currentConflict.value?.reason === "name" ? t("settings.types.importNameTitle") : t("settings.types.importOverwriteTitle"),
);
const conflictLead = computed(() => {
  const conflict = currentConflict.value;
  if (!conflict) {
    return "";
  }
  const kind = conflict.kind === "fieldGroup" ? t("settings.types.kindGroup") : t("settings.types.kindType");
  const key = conflict.reason === "name" ? "settings.types.importNameLead" : "settings.types.importOverwriteLead";
  return t(key, { kind, name: conflict.name });
});
const conflictConfirm = computed(() =>
  currentConflict.value?.reason === "name" ? t("settings.types.importRename") : t("settings.types.importOverwrite"),
);

function changeText(change: { field: string; before: string; after: string }): string {
  return t("settings.types.diffLine", {
    field: t(`settings.types.diff.${change.field}`),
    before: change.before || "–",
    after: change.after || "–",
  });
}

function openConflict() {
  const conflict = currentConflict.value;
  if (!conflict) {
    return;
  }
  renameName.value = conflict.suggestedName;
  conflictError.value = "";
  conflictSettled.value = false;
  conflictOpen.value = true;
}

function acceptConflict() {
  const conflict = currentConflict.value;
  if (!conflict) {
    return;
  }
  if (conflict.reason === "name") {
    if (!technicalNameOk(renameName.value)) {
      conflictError.value = t("settings.types.invalid");
      return;
    }
    decisions.value.push({
      kind: conflict.kind,
      id: conflict.id,
      action: "rename",
      name: renameName.value.trim(),
    });
  } else {
    decisions.value.push({ kind: conflict.kind, id: conflict.id, action: "overwrite" });
  }
  nextConflict();
}

function skipConflict() {
  const conflict = currentConflict.value;
  if (!conflict) {
    return;
  }
  decisions.value.push({ kind: conflict.kind, id: conflict.id, action: "skip" });
  nextConflict();
}

function nextConflict() {
  conflictSettled.value = true;
  if (conflictIndex.value + 1 >= conflicts.value.length) {
    conflictOpen.value = false;
    void finishImport();
    return;
  }
  conflictIndex.value += 1;
  openConflict();
}

function onConflictHide() {
  if (!conflictSettled.value) {
    pendingExchange.value = null;
    transferring.value = false;
  }
  conflictSettled.value = false;
}

async function finishImport() {
  const body = pendingExchange.value;
  if (!body) {
    transferring.value = false;
    return;
  }
  try {
    const result = await importDocumentTypes(body, decisions.value);
    pendingExchange.value = null;
    await load();
    noticeIsError.value = false;
    notice.value = t("settings.types.imported", { types: result.documentTypes, groups: result.fieldGroups });
  } catch (e) {
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    noticeIsError.value = true;
    notice.value = actionMessage(e instanceof Error ? e.message : "");
  } finally {
    transferring.value = false;
  }
}

async function exportCatalog() {
  notice.value = "";
  transferring.value = true;
  try {
    const data = await exportDocumentTypes();
    downloadJson(data, "document-types.json");
  } catch (e) {
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    noticeIsError.value = true;
    notice.value = actionMessage(e instanceof Error ? e.message : "");
  } finally {
    transferring.value = false;
  }
}

async function onImportFile(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) {
    return;
  }
  notice.value = "";
  let parsed: unknown;
  try {
    parsed = JSON.parse(await file.text());
  } catch {
    noticeIsError.value = true;
    notice.value = t("settings.types.importInvalid");
    return;
  }
  if (!isExchange(parsed)) {
    noticeIsError.value = true;
    notice.value = t("settings.types.importInvalid");
    return;
  }
  const body: DocumentTypeExchange = {
    documentTypes: parsed.documentTypes,
    fieldGroups: parsed.fieldGroups ?? [],
  };
  transferring.value = true;
  try {
    const found = await previewDocumentTypeImport(body);
    if (found.length === 0) {
      const result = await importDocumentTypes(body, []);
      await load();
      noticeIsError.value = false;
      notice.value = t("settings.types.imported", { types: result.documentTypes, groups: result.fieldGroups });
      return;
    }
    pendingExchange.value = body;
    conflicts.value = found;
    conflictIndex.value = 0;
    decisions.value = [];
    openConflict();
  } catch (e) {
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    noticeIsError.value = true;
    notice.value = actionMessage(e instanceof Error ? e.message : "");
  } finally {
    if (!conflictOpen.value) {
      transferring.value = false;
    }
  }
}

function openDelete() {
  if (!selected.value) {
    return;
  }
  deleteOpen.value = true;
}

async function submitDelete() {
  const docType = selected.value;
  if (!docType) {
    return;
  }
  deleting.value = true;
  error.value = "";
  try {
    await deleteDocumentType(docType.id);
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
        <span v-tooltip.bottom="editTip">
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
        <span v-tooltip.bottom="deleteTip">
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
        <Button
          type="button"
          icon="pi pi-ellipsis-h"
          text
          rounded
          :aria-label="t('settings.types.more')"
          v-tooltip.bottom="t('settings.types.more')"
          :disabled="transferring"
          @click="openMore"
        />
        <Menu ref="moreMenu" :model="moreItems" popup />
        <input ref="importInput" class="file-input" type="file" accept="application/json,.json" @change="onImportFile" />
      </div>
    </div>
    <p v-if="notice" :class="noticeIsError ? 'error' : 'notice'">{{ notice }}</p>
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
        <template #body="{ data }">{{ data.fieldGroups.length }}</template>
      </Column>
    </DataTable>

    <Dialog v-model:visible="editorOpen" modal :header="editorTitle" :style="{ width: 'min(56rem, 94vw)' }">
      <form class="form" @submit.prevent="submit">
        <p v-if="formError" class="error">{{ formError }}</p>
        <div class="pair">
          <label v-if="editingId">
            {{ t("settings.types.id") }}
            <input :value="editingId" readonly tabindex="-1" class="readonly" />
          </label>
          <label :class="{ span2: !editingId }">
            {{ t("settings.types.name") }}
            <input v-model="form.name" required autocomplete="off" />
          </label>
        </div>
        <div class="pair">
          <label>
            {{ t("settings.types.labelDe") }}
            <input v-model="form.labels.de" autocomplete="off" />
          </label>
          <label>
            {{ t("settings.types.labelEn") }}
            <input v-model="form.labels.en" autocomplete="off" />
          </label>
        </div>
        <div class="pair">
          <label>
            {{ t("settings.types.descriptionDe") }}
            <textarea v-model="form.description.de" rows="2" />
          </label>
          <label>
            {{ t("settings.types.descriptionEn") }}
            <textarea v-model="form.description.en" rows="2" />
          </label>
        </div>
        <fieldset class="groups">
          <legend>{{ t("settings.types.fieldGroups") }}</legend>
          <div v-for="(id, index) in form.fieldGroups" :key="id" class="group">
            <span class="name-cell">
              {{ groupLabel(id) }}
              <i
                v-if="isSystem(id)"
                class="pi pi-lock"
                :title="t('settings.types.systemGroup')"
                :aria-label="t('settings.types.systemGroup')"
              />
            </span>
            <span class="group-actions">
              <Button type="button" icon="pi pi-arrow-up" text rounded :aria-label="t('settings.types.moveUp')" :disabled="index === 0" @click="moveGroup(index, -1)" />
              <Button type="button" icon="pi pi-arrow-down" text rounded :aria-label="t('settings.types.moveDown')" :disabled="index === form.fieldGroups.length - 1" @click="moveGroup(index, 1)" />
              <span v-tooltip.bottom="isSystem(id) ? t('settings.types.systemGroup') : t('settings.types.removeGroup')">
                <Button
                  type="button"
                  icon="pi pi-times"
                  text
                  rounded
                  severity="danger"
                  :aria-label="t('settings.types.removeGroup')"
                  :disabled="isSystem(id)"
                  @click="removeGroup(index)"
                />
              </span>
            </span>
          </div>
          <div class="add-group">
            <select v-model="addGroupId">
              <option value="">{{ t("settings.types.chooseGroup") }}</option>
              <option v-for="group in availableGroups" :key="group.id" :value="group.id">
                {{ shownLabel(group) }}
              </option>
            </select>
            <Button type="button" :label="t('settings.types.addGroup')" text :disabled="!addGroupId" @click="addGroup" />
          </div>
        </fieldset>
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

    <Dialog
      v-model:visible="conflictOpen"
      modal
      :header="conflictTitle"
      :style="{ width: '32rem' }"
      @hide="onConflictHide"
    >
      <p>{{ conflictLead }}</p>
      <ul v-if="currentConflict && currentConflict.changes.length" class="diff">
        <li v-for="change in currentConflict.changes" :key="change.field">{{ changeText(change) }}</li>
      </ul>
      <p v-else>{{ t("settings.types.importNoChanges") }}</p>
      <label v-if="currentConflict?.reason === 'name'" class="rename">
        {{ t("settings.types.importNewName") }}
        <input v-model="renameName" autocomplete="off" />
      </label>
      <p v-if="conflictError" class="error">{{ conflictError }}</p>
      <template #footer>
        <Button type="button" :label="t('settings.types.importSkip')" text @click="skipConflict" />
        <Button type="button" :label="conflictConfirm" @click="acceptConflict" />
      </template>
    </Dialog>

    <Dialog v-model:visible="deleteOpen" modal :header="t('settings.types.deleteTitle')" :style="{ width: '24rem' }">
      <p>{{ t("settings.types.deleteConfirm") }}</p>
      <template #footer>
        <Button type="button" :label="t('settings.types.cancel')" text @click="deleteOpen = false" />
        <Button type="button" severity="danger" :label="t('settings.types.delete')" :loading="deleting" @click="submitDelete" />
      </template>
    </Dialog>
  </SettingsPage>
</template>

<style scoped>
.name-cell {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  min-width: 0;
}
.name-cell .pi-lock {
  color: #5b6570;
  font-size: 0.85rem;
}
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
  margin: 0 0 0.75rem;
  color: #9b2c2c;
}
.notice {
  margin: 0 0 0.75rem;
  color: #1f6b3a;
}
.file-input {
  display: none;
}
.diff {
  margin: 0.75rem 0 0;
  padding-left: 1.1rem;
}
.diff li {
  margin: 0.25rem 0;
}
.rename {
  display: grid;
  gap: 0.3rem;
  margin-top: 0.85rem;
}
.rename input {
  padding: 0.45rem 0.6rem;
  border: 1px solid #c9d0d6;
  border-radius: 8px;
  font: inherit;
}
.form {
  display: grid;
  gap: 0.75rem;
}
.pair {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}
.pair > .span2 {
  grid-column: 1 / -1;
}
.form input.readonly {
  background: #f3f5f7;
  color: #5a6672;
  font-family: ui-monospace, monospace;
  font-size: 0.9rem;
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
.groups {
  margin: 0;
  padding: 0;
  border: 0;
  display: grid;
  gap: 0.45rem;
}
.group {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.35rem 0.6rem;
  border: 1px solid #e4e8ec;
  border-radius: 10px;
  background: #f8f9fb;
}
.group-actions {
  display: flex;
  align-items: center;
  flex: 0 0 auto;
}
.add-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.add-group select {
  flex: 1;
}
</style>
