<script setup lang="ts">
import SettingsPage from "../SettingsPage.vue";

defineProps<{ title: string }>();
</script>

<template>
  <SettingsPage :title="title" />
</template>
