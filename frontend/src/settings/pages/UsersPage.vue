<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import Button from "primevue/button";
import Column from "primevue/column";
import DataTable, { type DataTablePageEvent, type DataTableSortEvent } from "primevue/datatable";
import Dialog from "primevue/dialog";
import {
  createUser,
  deleteUser,
  listUsers,
  resetUserPassword,
  startAuthorization,
  updateUser,
  type SettingsUser,
} from "../../auth/oidc";
import SettingsPage from "../SettingsPage.vue";

defineProps<{ title: string }>();

const roles = ["admin", "archivist", "clerk", "reader"] as const;

const { t, locale } = useI18n();
const users = ref<SettingsUser[]>([]);
const selected = ref<SettingsUser[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = ref(10);
const sortField = ref("username");
const sortOrder = ref<1 | -1>(1);
const filterText = ref("");
const appliedPrefix = ref("");
const loading = ref(false);
const error = ref("");

const createOpen = ref(false);
const creating = ref(false);
const createdPassword = ref("");
const formError = ref("");
const form = ref(emptyForm());

const deleteOpen = ref(false);
const deleting = ref(false);

const editOpen = ref(false);
const editing = ref(false);
const resetting = ref(false);
const editId = ref("");
const editError = ref("");
const editSaved = ref("");
const resetPassword = ref("");

let filterTimer = 0;

function emptyForm() {
  return {
    username: "",
    firstName: "",
    lastName: "",
    email: "",
    roles: ["reader"] as string[],
  };
}

function roleLabel(role: string): string {
  switch (role) {
    case "admin":
      return t("settings.users.roleAdmin");
    case "archivist":
      return t("settings.users.roleArchivist");
    case "clerk":
      return t("settings.users.roleClerk");
    default:
      return t("settings.users.roleReader");
  }
}

function formatLastLogin(value: string | null | undefined): string {
  if (!value) {
    return t("account.lastLoginNever");
  }
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) {
    return t("account.lastLoginNever");
  }
  return new Intl.DateTimeFormat(locale.value === "de" ? "de-DE" : "en-GB", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(d);
}

function activePrefix(value: string): string {
  const trimmed = value.trim();
  return [...trimmed].length >= 3 ? trimmed : "";
}

function actionMessage(key: string): string {
  switch (key) {
    case "already-exists":
      return t("settings.users.alreadyExists");
    case "invalid-user":
      return t("settings.users.invalidUser");
    case "delete-self":
      return t("settings.users.deleteSelf");
    case "last-admin":
      return t("settings.users.lastAdmin");
    case "forbidden":
      return t("settings.users.forbidden");
    default:
      return t("settings.users.actionError");
  }
}

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const result = await listUsers(
      page.value,
      pageSize.value,
      sortField.value,
      sortOrder.value === -1 ? "desc" : "asc",
      appliedPrefix.value,
    );
    users.value = result.items;
    total.value = result.total;
    const ids = new Set(result.items.map((user) => user.id));
    selected.value = selected.value.filter((user) => ids.has(user.id));
  } catch (e) {
    users.value = [];
    total.value = 0;
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    if (e instanceof Error && e.message === "forbidden") {
      error.value = t("settings.users.forbidden");
      return;
    }
    error.value = t("settings.users.loadError");
  } finally {
    loading.value = false;
  }
}

function onPage(event: DataTablePageEvent) {
  page.value = event.page + 1;
  pageSize.value = event.rows;
  void load();
}

function onRowDoubleClick(event: { data: SettingsUser }) {
  selected.value = [event.data];
  openEdit();
}

function onSort(event: DataTableSortEvent) {
  sortField.value = typeof event.sortField === "string" && event.sortField ? event.sortField : "username";
  sortOrder.value = event.sortOrder === -1 ? -1 : 1;
  page.value = 1;
  void load();
}

function onFilterInput() {
  window.clearTimeout(filterTimer);
  filterTimer = window.setTimeout(() => {
    const next = activePrefix(filterText.value);
    if (next === appliedPrefix.value) {
      return;
    }
    appliedPrefix.value = next;
    page.value = 1;
    void load();
  }, 250);
}

function openCreate() {
  form.value = emptyForm();
  formError.value = "";
  createdPassword.value = "";
  createOpen.value = true;
}

function toggleRole(role: string, checked: boolean) {
  const current = new Set(form.value.roles);
  if (checked) {
    current.add(role);
  } else {
    current.delete(role);
  }
  form.value.roles = [...current];
}

async function submitCreate() {
  formError.value = "";
  if (!form.value.username.trim() || form.value.roles.length === 0) {
    formError.value = form.value.roles.length === 0 ? t("settings.users.rolesRequired") : t("settings.users.invalidUser");
    return;
  }
  creating.value = true;
  try {
    const result = await createUser({
      username: form.value.username.trim(),
      firstName: form.value.firstName.trim(),
      lastName: form.value.lastName.trim(),
      email: form.value.email.trim(),
      roles: form.value.roles,
    });
    createdPassword.value = result.password;
    page.value = 1;
    await load();
  } catch (e) {
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    formError.value = actionMessage(e instanceof Error ? e.message : "");
  } finally {
    creating.value = false;
  }
}

function openEdit() {
  const user = selected.value[0];
  if (!user || selected.value.length !== 1) {
    return;
  }
  form.value = {
    username: user.username,
    firstName: user.firstName ?? "",
    lastName: user.lastName ?? "",
    email: user.email ?? "",
    roles: [...user.roles],
  };
  editId.value = user.id;
  editError.value = "";
  editSaved.value = "";
  resetPassword.value = "";
  editOpen.value = true;
}

async function submitEdit() {
  editError.value = "";
  editSaved.value = "";
  if (!form.value.username.trim() || form.value.roles.length === 0) {
    editError.value = form.value.roles.length === 0 ? t("settings.users.rolesRequired") : t("settings.users.invalidUser");
    return;
  }
  editing.value = true;
  try {
    await updateUser(editId.value, {
      username: form.value.username.trim(),
      firstName: form.value.firstName.trim(),
      lastName: form.value.lastName.trim(),
      email: form.value.email.trim(),
      roles: form.value.roles,
    });
    editOpen.value = false;
    await load();
  } catch (e) {
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    editError.value = actionMessage(e instanceof Error ? e.message : "");
  } finally {
    editing.value = false;
  }
}

async function submitReset() {
  editError.value = "";
  resetting.value = true;
  try {
    const result = await resetUserPassword(editId.value);
    resetPassword.value = result.password;
    await load();
  } catch (e) {
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    editError.value = actionMessage(e instanceof Error ? e.message : "");
  } finally {
    resetting.value = false;
  }
}

function openDelete() {
  if (!selected.value.length) {
    return;
  }
  deleteOpen.value = true;
}

async function submitDelete() {
  deleting.value = true;
  error.value = "";
  try {
    for (const user of selected.value) {
      await deleteUser(user.id);
    }
    selected.value = [];
    deleteOpen.value = false;
    await load();
  } catch (e) {
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    error.value = actionMessage(e instanceof Error ? e.message : "");
    deleteOpen.value = false;
    await load();
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
      <input
        v-model="filterText"
        type="search"
        class="filter"
        :placeholder="t('settings.users.filterPlaceholder')"
        :aria-label="t('settings.users.filterPlaceholder')"
        @input="onFilterInput"
      />
      <div class="actions">
        <Button
          type="button"
          icon="pi pi-user-plus"
          text
          rounded
          :aria-label="t('settings.users.create')"
          v-tooltip.bottom="t('settings.users.create')"
          @click="openCreate"
        />
        <span v-tooltip.bottom="selected.length === 1 ? t('settings.users.edit') : t('settings.users.editNeedSelection')">
          <Button
            type="button"
            icon="pi pi-user-edit"
            text
            rounded
            :aria-label="t('settings.users.edit')"
            :disabled="selected.length !== 1"
            @click="openEdit"
          />
        </span>
        <span v-tooltip.bottom="selected.length ? t('settings.users.delete') : t('settings.users.deleteNeedSelection')">
          <Button
            type="button"
            icon="pi pi-trash"
            text
            rounded
            severity="danger"
            :aria-label="t('settings.users.delete')"
            :disabled="selected.length === 0 || deleting"
            @click="openDelete"
          />
        </span>
      </div>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
    <DataTable
      v-else
      v-model:selection="selected"
      :value="users"
      data-key="id"
      lazy
      paginator
      :rows="pageSize"
      :first="(page - 1) * pageSize"
      :total-records="total"
      :loading="loading"
      :rows-per-page-options="[10, 20, 50]"
      sort-mode="single"
      :sort-field="sortField"
      :sort-order="sortOrder"
      size="small"
      @page="onPage"
      @sort="onSort"
      @row-dblclick="onRowDoubleClick"
    >
      <template #empty>{{ t("settings.users.empty") }}</template>
      <Column selection-mode="multiple" header-style="width: 3rem" />
      <Column field="username" :header="t('settings.users.username')" sortable />
      <Column field="firstName" :header="t('settings.users.firstName')" sortable>
        <template #body="{ data }">{{ data.firstName || "—" }}</template>
      </Column>
      <Column field="lastName" :header="t('settings.users.lastName')" sortable>
        <template #body="{ data }">{{ data.lastName || "—" }}</template>
      </Column>
      <Column field="email" :header="t('settings.users.email')" sortable>
        <template #body="{ data }">{{ data.email || "—" }}</template>
      </Column>
      <Column field="roles" :header="t('settings.users.roles')" sortable>
        <template #body="{ data }">{{ data.roles.join(", ") || "—" }}</template>
      </Column>
      <Column field="lastLogin" :header="t('settings.users.lastLogin')" sortable>
        <template #body="{ data }">{{ formatLastLogin(data.lastLogin) }}</template>
      </Column>
      <Column field="mustChangePassword" :header="t('settings.users.mustChange')" sortable>
        <template #body="{ data }">{{ data.mustChangePassword ? t("settings.users.yes") : t("settings.users.no") }}</template>
      </Column>
    </DataTable>

    <Dialog v-model:visible="createOpen" modal :header="t('settings.users.createTitle')" :style="{ width: '26rem' }">
      <p v-if="createdPassword" class="hint">{{ t("settings.users.passwordHint") }}</p>
      <p v-if="createdPassword" class="password">
        <span>{{ t("settings.users.passwordOnce") }}</span>
        <code>{{ createdPassword }}</code>
      </p>
      <form v-else class="form" @submit.prevent="submitCreate">
        <label>
          {{ t("settings.users.username") }}
          <input v-model="form.username" required autocomplete="off" />
        </label>
        <label>
          {{ t("settings.users.firstName") }}
          <input v-model="form.firstName" autocomplete="off" />
        </label>
        <label>
          {{ t("settings.users.lastName") }}
          <input v-model="form.lastName" autocomplete="off" />
        </label>
        <label>
          {{ t("settings.users.email") }}
          <input v-model="form.email" type="email" autocomplete="off" />
        </label>
        <fieldset>
          <legend>{{ t("settings.users.roles") }}</legend>
          <label v-for="role in roles" :key="role" class="check">
            <input
              type="checkbox"
              :checked="form.roles.includes(role)"
              @change="toggleRole(role, ($event.target as HTMLInputElement).checked)"
            />
            {{ roleLabel(role) }}
          </label>
        </fieldset>
        <p v-if="formError" class="error">{{ formError }}</p>
      </form>
      <template #footer>
        <Button v-if="createdPassword" type="button" :label="t('settings.users.close')" @click="createOpen = false" />
        <template v-else>
          <Button type="button" :label="t('settings.users.cancel')" text @click="createOpen = false" />
          <Button type="button" :label="t('settings.users.createSubmit')" :loading="creating" @click="submitCreate" />
        </template>
      </template>
    </Dialog>

    <Dialog v-model:visible="editOpen" modal :header="t('settings.users.editTitle')" :style="{ width: '26rem' }">
      <form class="form" @submit.prevent="submitEdit">
        <label>
          {{ t("settings.users.username") }}
          <input v-model="form.username" required autocomplete="off" />
        </label>
        <label>
          {{ t("settings.users.firstName") }}
          <input v-model="form.firstName" autocomplete="off" />
        </label>
        <label>
          {{ t("settings.users.lastName") }}
          <input v-model="form.lastName" autocomplete="off" />
        </label>
        <label>
          {{ t("settings.users.email") }}
          <input v-model="form.email" type="email" autocomplete="off" />
        </label>
        <fieldset>
          <legend>{{ t("settings.users.roles") }}</legend>
          <label v-for="role in roles" :key="role" class="check">
            <input
              type="checkbox"
              :checked="form.roles.includes(role)"
              @change="toggleRole(role, ($event.target as HTMLInputElement).checked)"
            />
            {{ roleLabel(role) }}
          </label>
        </fieldset>
        <p v-if="editSaved" class="ok">{{ editSaved }}</p>
        <p v-if="resetPassword" class="hint">{{ t("settings.users.passwordHint") }}</p>
        <p v-if="resetPassword" class="password">
          <span>{{ t("settings.users.passwordOnce") }}</span>
          <code>{{ resetPassword }}</code>
        </p>
        <p v-if="editError" class="error">{{ editError }}</p>
      </form>
      <template #footer>
        <Button type="button" :label="t('settings.users.resetPassword')" text :loading="resetting" @click="submitReset" />
        <Button type="button" :label="t('settings.users.cancel')" text @click="editOpen = false" />
        <Button type="button" :label="t('settings.users.editSubmit')" :loading="editing" @click="submitEdit" />
      </template>
    </Dialog>

    <Dialog v-model:visible="deleteOpen" modal :header="t('settings.users.deleteTitle')" :style="{ width: '24rem' }">
      <p>{{ t("settings.users.deleteConfirm") }}</p>
      <ul>
        <li v-for="user in selected" :key="user.id">{{ user.username }}</li>
      </ul>
      <template #footer>
        <Button type="button" :label="t('settings.users.cancel')" text @click="deleteOpen = false" />
        <Button type="button" severity="danger" :label="t('settings.users.delete')" :loading="deleting" @click="submitDelete" />
      </template>
    </Dialog>
  </SettingsPage>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin: 0 0 0.85rem;
}
.filter {
  width: min(18rem, 100%);
  box-sizing: border-box;
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
.actions {
  display: flex;
  align-items: center;
  gap: 0.15rem;
}
.error {
  margin: 0 0 0.75rem;
  color: #9b2c2c;
}
.ok {
  margin: 0;
  color: #1b7f3a;
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
.form input[type="text"],
.form input[type="email"],
.form input:not([type="checkbox"]) {
  padding: 0.45rem 0.6rem;
  border: 1px solid #c9d0d6;
  border-radius: 8px;
  font: inherit;
}
fieldset {
  margin: 0;
  padding: 0;
  border: 0;
}
.check {
  display: flex !important;
  align-items: center;
  gap: 0.45rem;
  margin-top: 0.35rem;
}
.hint,
.password {
  margin: 0 0 0.75rem;
}
.password {
  display: grid;
  gap: 0.35rem;
}
.password code {
  font-family: ui-monospace, monospace;
  font-size: 0.95rem;
}
ul {
  margin: 0.5rem 0 0;
  padding-left: 1.2rem;
}
</style>
