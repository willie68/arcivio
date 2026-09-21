<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { changePassword, hasPendingAuthRequest, login, startAuthorization } from "../auth/oidc";
import lockupLight from "../assets/brand/lockup-light.png";

const { t } = useI18n();
const username = ref("admin");
const password = ref("");
const oldPassword = ref("");
const newPassword = ref("");
const confirmPassword = ref("");
const needChange = ref(false);
const error = ref("");
const busy = ref(false);
const preparing = ref(false);

function applyResult(res: { status: string; redirectTo?: string }) {
  if (res.status === "password_change_required") {
    needChange.value = true;
    oldPassword.value = password.value;
    error.value = t("login.mustChange");
    return;
  }
  if (res.status === "ok" && res.redirectTo) {
    window.location.assign(res.redirectTo);
    return;
  }
  error.value = t("login.unexpected");
}

onMounted(async () => {
  if (hasPendingAuthRequest()) {
    return;
  }
  preparing.value = true;
  try {
    await startAuthorization();
  } catch (e) {
    preparing.value = false;
    error.value = e instanceof Error ? e.message : t("login.startFailed");
  }
});

async function submitLogin() {
  error.value = "";
  busy.value = true;
  try {
    const res = await login(username.value, password.value);
    applyResult(res);
  } catch (e) {
    const msg = e instanceof Error ? e.message : t("login.failed");
    if (msg.includes("start the authorization") || msg.includes("login-required")) {
      await startAuthorization();
      return;
    }
    error.value = msg;
  } finally {
    busy.value = false;
  }
}

async function submitChange() {
  error.value = "";
  if (newPassword.value !== confirmPassword.value) {
    error.value = t("changePassword.mismatch");
    return;
  }
  busy.value = true;
  try {
    const res = await changePassword(oldPassword.value, newPassword.value);
    applyResult(res);
  } catch (e) {
    error.value = e instanceof Error ? e.message : t("changePassword.failed");
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <main class="page">
    <section class="card">
      <img class="brand" :src="lockupLight" :alt="t('brand.name')" />
      <p class="lead">{{ t("login.title") }}</p>
      <p v-if="preparing" class="hint">{{ t("login.preparing") }}</p>
      <form v-if="!needChange" class="form" @submit.prevent="submitLogin">
        <label>
          {{ t("login.username") }}
          <input v-model="username" type="text" autocomplete="username" class="native" />
        </label>
        <label>
          {{ t("login.password") }}
          <input v-model="password" type="password" autocomplete="current-password" class="native" />
        </label>
        <p v-if="error" class="error">{{ error }}</p>
        <button class="submit" type="submit" :disabled="busy">{{ t("login.submit") }}</button>
      </form>
      <form v-else class="form" @submit.prevent="submitChange">
        <p>{{ t("changePassword.intro") }}</p>
        <label>
          {{ t("changePassword.current") }}
          <input v-model="oldPassword" type="password" autocomplete="current-password" class="native" />
        </label>
        <label>
          {{ t("changePassword.next") }}
          <input v-model="newPassword" type="password" autocomplete="new-password" class="native" />
        </label>
        <label>
          {{ t("changePassword.confirm") }}
          <input v-model="confirmPassword" type="password" autocomplete="new-password" class="native" />
        </label>
        <p v-if="error" class="error">{{ error }}</p>
        <button class="submit" type="submit" :disabled="busy">{{ t("changePassword.submit") }}</button>
      </form>
    </section>
  </main>
</template>

<style scoped>
.page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  font-family: system-ui, sans-serif;
  background: #f4f6f8;
  color: #1b1f24;
}
.card {
  width: min(24rem, calc(100vw - 2rem));
  background: #fff;
  padding: 1.75rem;
  border-radius: 12px;
  box-shadow: 0 8px 30px rgb(0 0 0 / 8%);
}
.brand {
  display: block;
  width: min(16rem, 100%);
  height: auto;
  margin: 0 auto 0.25rem;
}
.lead {
  margin: 0 0 0.5rem;
  color: #555;
  text-align: center;
}
.hint {
  color: #555;
  font-size: 0.9rem;
}
.form {
  display: grid;
  gap: 0.85rem;
}
label {
  display: grid;
  gap: 0.35rem;
  font-size: 0.9rem;
}
.native,
.submit {
  width: 100%;
  box-sizing: border-box;
  padding: 0.55rem 0.7rem;
  border: 1px solid #c9d0d6;
  border-radius: 6px;
  font: inherit;
}
.submit {
  background: #1f4b99;
  color: #fff;
  border-color: #1f4b99;
  cursor: pointer;
}
.submit:disabled {
  opacity: 0.7;
  cursor: wait;
}
.error {
  color: #b42318;
  margin: 0;
}
</style>
