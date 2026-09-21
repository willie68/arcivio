<script setup lang="ts">
defineProps<{ title: string }>();
</script>

<template>
  <section class="settings-page">
    <h1>{{ title }}</h1>
    <slot />
  </section>
</template>

<style scoped>
.settings-page {
  max-width: 44rem;
}
h1 {
  margin: 0 0 1rem;
  font-size: 1.35rem;
  font-weight: 650;
}
</style>
