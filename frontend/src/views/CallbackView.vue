<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { exchangeCode, startAuthorization } from "../auth/oidc";

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
    error.value = e instanceof Error ? e.message : "Anmeldung fehlgeschlagen";
  }
});
</script>

<template>
  <main style="font-family: system-ui, sans-serif; padding: 2rem">
    <h1>Arcivio</h1>
    <p v-if="!error">Anmeldung wird abgeschlossen …</p>
    <p v-else style="color: #b42318">{{ error }}</p>
  </main>
</template>
