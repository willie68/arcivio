<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { fetchMe, startAuthorization } from "../auth/oidc";

const { t } = useI18n();
const username = ref("");
const roles = ref<string[]>([]);
const error = ref("");

onMounted(async () => {
  try {
    const me = await fetchMe();
    username.value = me.username;
    roles.value = me.roles;
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
  <section class="home">
    <h1>{{ t("brand.name") }}</h1>
    <p>{{ t("home.tagline") }}</p>
    <p v-if="username">
      {{ t("home.signedIn", { name: username }) }}
      <span v-if="roles.length"> ({{ roles.join(", ") }})</span>
    </p>
    <p v-if="error" class="error">{{ error }}</p>
  </section>
</template>

<style scoped>
.home {
  max-width: 40rem;
}
h1 {
  margin: 0 0 0.35rem;
}
.error {
  color: #b42318;
}
</style>
