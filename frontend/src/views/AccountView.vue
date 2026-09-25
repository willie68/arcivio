<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { fetchMe, startAuthorization, updateProfile } from "../auth/oidc";

const { t, locale } = useI18n();
const username = ref("");
const firstName = ref("");
const lastName = ref("");
const email = ref("");
const roles = ref<string[]>([]);
const id = ref("");
const lastLogin = ref<string | null>(null);
const error = ref("");
const busy = ref(false);
const editing = ref(false);

const emit = defineEmits<{ close: [] }>();

function shown(value: string): string {
  return value.trim() ? value : "—";
}

function formatLastLogin(value: string | null): string {
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

onMounted(async () => {
  try {
    const me = await fetchMe();
    username.value = me.username;
    firstName.value = me.firstName ?? "";
    lastName.value = me.lastName ?? "";
    email.value = me.email ?? "";
    roles.value = me.roles;
    id.value = me.id;
    lastLogin.value = me.lastLogin ?? null;
  } catch (e) {
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    error.value = e instanceof Error ? e.message : t("home.profileUnavailable");
  }
});

function startEdit() {
  error.value = "";
  editing.value = true;
}

async function save() {
  error.value = "";
  busy.value = true;
  try {
    const me = await updateProfile({
      firstName: firstName.value.trim(),
      lastName: lastName.value.trim(),
      email: email.value.trim(),
    });
    firstName.value = me.firstName ?? "";
    lastName.value = me.lastName ?? "";
    email.value = me.email ?? "";
    editing.value = false;
  } catch (e) {
    if (e instanceof Error && e.message === "not authenticated") {
      await startAuthorization();
      return;
    }
    error.value = e instanceof Error && e.message === "invalid-user" ? t("settings.users.invalidUser") : t("settings.users.actionError");
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <section class="account">
    <form class="form" @submit.prevent="save">
      <div>
        <span class="label">{{ t("account.username") }}</span>
        <p>{{ username }}</p>
      </div>
      <label v-if="editing">
        {{ t("account.firstName") }}
        <input v-model="firstName" autocomplete="given-name" />
      </label>
      <div v-else>
        <span class="label">{{ t("account.firstName") }}</span>
        <p>{{ shown(firstName) }}</p>
      </div>
      <label v-if="editing">
        {{ t("account.lastName") }}
        <input v-model="lastName" autocomplete="family-name" />
      </label>
      <div v-else>
        <span class="label">{{ t("account.lastName") }}</span>
        <p>{{ shown(lastName) }}</p>
      </div>
      <label v-if="editing">
        {{ t("account.email") }}
        <input v-model="email" type="email" autocomplete="email" />
      </label>
      <div v-else>
        <span class="label">{{ t("account.email") }}</span>
        <p>{{ shown(email) }}</p>
      </div>
      <div>
        <span class="label">{{ t("account.roles") }}</span>
        <p>{{ roles.join(", ") || "—" }}</p>
      </div>
      <div>
        <span class="label">{{ t("account.id") }}</span>
        <p class="mono">{{ id }}</p>
      </div>
      <div>
        <span class="label">{{ t("account.lastLogin") }}</span>
        <p>{{ formatLastLogin(lastLogin) }}</p>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
      <div class="actions">
        <button v-if="editing" type="submit" :disabled="busy || !username">{{ t("account.save") }}</button>
        <button v-else type="button" @click="startEdit">{{ t("account.edit") }}</button>
        <button type="button" class="ghost" @click="emit('close')">{{ t("account.close") }}</button>
      </div>
    </form>
  </section>
</template>

<style scoped>
.account {
  max-width: 26rem;
}
.form {
  display: grid;
  gap: 0.85rem;
}
.label,
label {
  color: #5b6570;
  font-size: 0.85rem;
}
label {
  display: grid;
  gap: 0.3rem;
}
p {
  margin: 0.2rem 0 0;
  color: #1b1f24;
}
input {
  width: 100%;
  box-sizing: border-box;
  padding: 0.55rem 0.7rem;
  border: 1px solid #c9d0d6;
  border-radius: 6px;
  font: inherit;
  color: #1b1f24;
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 0.35rem;
}
button {
  padding: 0.5rem 0.85rem;
  border: 1px solid #1f4b99;
  border-radius: 6px;
  background: #1f4b99;
  color: #fff;
  font: inherit;
  cursor: pointer;
}
button.ghost {
  background: transparent;
  color: #1f4b99;
}
button:disabled {
  opacity: 0.7;
  cursor: wait;
}
.mono {
  font-family: ui-monospace, monospace;
  font-size: 0.9rem;
  word-break: break-all;
}
.error {
  color: #b42318;
}
</style>
