<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { changeOwnPassword, startAuthorization } from "../auth/oidc";

const { t } = useI18n();
const oldPassword = ref("");
const newPassword = ref("");
const confirmPassword = ref("");
const error = ref("");
const success = ref("");
const busy = ref(false);

async function submit() {
  error.value = "";
  success.value = "";
  if (newPassword.value !== confirmPassword.value) {
    error.value = t("changePassword.mismatch");
    return;
  }
  busy.value = true;
  try {
    await changeOwnPassword(oldPassword.value, newPassword.value);
    oldPassword.value = "";
    newPassword.value = "";
    confirmPassword.value = "";
    success.value = t("changePasswordLoggedIn.success");
  } catch (e) {
    const msg = e instanceof Error ? e.message : "";
    if (msg === "not authenticated") {
      await startAuthorization();
      return;
    }
    if (msg.includes("old password") || msg.includes("invalid-password")) {
      error.value = t("changePasswordLoggedIn.wrongOld");
    } else {
      error.value = msg || t("changePassword.failed");
    }
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <section class="page">
    <p class="lead">{{ t("changePasswordLoggedIn.intro") }}</p>
    <form class="form" @submit.prevent="submit">
      <label>
        {{ t("changePassword.current") }}
        <input v-model="oldPassword" type="password" autocomplete="current-password" />
      </label>
      <label>
        {{ t("changePassword.next") }}
        <input v-model="newPassword" type="password" autocomplete="new-password" />
      </label>
      <label>
        {{ t("changePassword.confirm") }}
        <input v-model="confirmPassword" type="password" autocomplete="new-password" />
      </label>
      <p v-if="error" class="error">{{ error }}</p>
      <p v-if="success" class="ok">{{ success }}</p>
      <button type="submit" :disabled="busy">{{ t("changePassword.submit") }}</button>
    </form>
  </section>
</template>

<style scoped>
.page {
  max-width: 22rem;
}
.lead {
  color: #5b6570;
  margin: 0 0 1rem;
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
input,
button {
  width: 100%;
  box-sizing: border-box;
  padding: 0.55rem 0.7rem;
  border: 1px solid #c9d0d6;
  border-radius: 6px;
  font: inherit;
}
button {
  background: #1f4b99;
  color: #fff;
  border-color: #1f4b99;
  cursor: pointer;
}
button:disabled {
  opacity: 0.7;
  cursor: wait;
}
.error {
  color: #b42318;
  margin: 0;
}
.ok {
  color: #1b7f3a;
  margin: 0;
}
</style>
