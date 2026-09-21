<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { NavNode } from "./catalog";
import SettingsTreeNode from "./SettingsTreeNode.vue";

const props = defineProps<{
  nodes: NavNode[];
  selectedId: string;
}>();

const emit = defineEmits<{
  select: [node: NavNode];
}>();

const { t } = useI18n();
const filter = ref("");

const visibleNodes = computed(() => filterNodes(props.nodes, filter.value.trim().toLowerCase()));

function filterNodes(nodes: NavNode[], needle: string): NavNode[] {
  if (!needle) {
    return nodes;
  }
  const out: NavNode[] = [];
  for (const node of nodes) {
    const labelMatch = node.label.toLowerCase().includes(needle);
    const children = node.children?.length ? filterNodes(node.children, needle) : [];
    if (labelMatch) {
      out.push(node);
    } else if (children.length) {
      out.push({ ...node, children });
    }
  }
  return out;
}
</script>

<template>
  <div class="tree">
    <input
      v-model="filter"
      type="search"
      class="filter"
      :placeholder="t('settings.filterPlaceholder')"
      :aria-label="t('settings.filterPlaceholder')"
    />
    <ul v-if="visibleNodes.length" class="list" role="tree">
      <SettingsTreeNode :nodes="visibleNodes" :selected-id="selectedId" @select="emit('select', $event)" />
    </ul>
    <p v-else class="empty">{{ t("settings.filterEmpty") }}</p>
  </div>
</template>

<style scoped>
.tree {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  min-height: 0;
  height: 100%;
}
.filter {
  width: 100%;
  box-sizing: border-box;
  padding: 0.45rem 0.65rem;
  border: 1px solid #c9d0d6;
  border-radius: 8px;
  background: #fff;
  font: inherit;
}
.filter:focus {
  outline: 2px solid #1f4b99;
  outline-offset: 1px;
}
.list,
.list :deep(ul) {
  margin: 0;
  padding: 0;
}
.empty {
  margin: 0.35rem 0 0;
  color: #5b6570;
  font-size: 0.9rem;
}
</style>
