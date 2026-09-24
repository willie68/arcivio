<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { fetchMe, startAuthorization } from "../auth/oidc";

const { t, locale } = useI18n();
const username = ref("");
const firstName = ref("");
const lastName = ref("");
const email = ref("");
const roles = ref<string[]>([]);
const id = ref("");
const lastLogin = ref<string | null>(null);
const error = ref("");

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
</script>

<template>
  <section class="account">
    <p v-if="error" class="error">{{ error }}</p>
    <dl v-else-if="username">
      <div>
        <dt>{{ t("account.username") }}</dt>
        <dd>{{ username }}</dd>
      </div>
      <div>
        <dt>{{ t("account.firstName") }}</dt>
        <dd>{{ firstName || "—" }}</dd>
      </div>
      <div>
        <dt>{{ t("account.lastName") }}</dt>
        <dd>{{ lastName || "—" }}</dd>
      </div>
      <div>
        <dt>{{ t("account.email") }}</dt>
        <dd>{{ email || "—" }}</dd>
      </div>
      <div>
        <dt>{{ t("account.roles") }}</dt>
        <dd>{{ roles.join(", ") || "—" }}</dd>
      </div>
      <div>
        <dt>{{ t("account.id") }}</dt>
        <dd class="mono">{{ id }}</dd>
      </div>
      <div>
        <dt>{{ t("account.lastLogin") }}</dt>
        <dd>{{ formatLastLogin(lastLogin) }}</dd>
      </div>
    </dl>
  </section>
</template>

<style scoped>
.account {
  max-width: 26rem;
}
dl {
  display: grid;
  gap: 0.85rem;
  margin: 0;
}
dt {
  color: #5b6570;
  font-size: 0.85rem;
}
dd {
  margin: 0.2rem 0 0;
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
