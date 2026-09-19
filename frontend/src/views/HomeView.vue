<script setup lang="ts">
import { onMounted, ref } from "vue";
import Button from "primevue/button";
import { fetchMe, logout, startAuthorization } from "../auth/oidc";

const username = ref("");
const roles = ref<string[]>([]);
const error = ref("");

onMounted(async () => {
  try {
    const me = await fetchMe();
    username.value = me.username;
    roles.value = me.roles;
  } catch {
    await startAuthorization();
  }
});

async function doLogout() {
  await logout();
  await startAuthorization();
}
</script>

<template>
  <main style="font-family: system-ui, sans-serif; padding: 2rem; max-width: 40rem">
    <h1>Arcivio</h1>
    <p>SOHO Dokumentenmanagement</p>
    <p v-if="username">
      Angemeldet als <strong>{{ username }}</strong>
      <span v-if="roles.length"> ({{ roles.join(", ") }})</span>
    </p>
    <p v-if="error">{{ error }}</p>
    <p>
      <a href="/swagger/">Swagger UI</a>
      ·
      <a href="/livez">livez</a>
      ·
      <a href="/readyz">readyz</a>
    </p>
    <Button label="Abmelden" @click="doLogout" />
  </main>
</template>
