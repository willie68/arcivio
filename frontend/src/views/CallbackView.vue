<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { exchangeCode, startAuthorization } from "../auth/oidc";
import lockupLight from "../assets/brand/lockup-light.png";

const { t } = useI18n();
const error = ref("");
const router = useRouter();

onMounted(async () => {
  const params = new URLSearchParams(window.location.search);
  const err = params.get("error");
  if (err) {
    error.value = params.get("error_description") || err;
    return;
  }
  const code = params.get("code");
  const state = params.get("state");
  if (!code || !state) {
    await startAuthorization();
    return;
  }
  try {
    await exchangeCode(code, state);
    await router.replace("/");
  } catch (e) {
    error.value = e instanceof Error ? e.message : t("callback.failed");
  }
});
</script>

<template>
  <main class="page">
    <img class="brand" :src="lockupLight" :alt="t('brand.name')" />
    <p v-if="!error">{{ t("callback.finishing") }}</p>
    <p v-else class="error">{{ error }}</p>
  </main>
</template>

<style scoped>
.page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 0.75rem;
  font-family: system-ui, sans-serif;
  padding: 2rem;
}
.brand {
  width: min(16rem, 100%);
  height: auto;
}
.error {
  color: #b42318;
  margin: 0;
}
</style>
