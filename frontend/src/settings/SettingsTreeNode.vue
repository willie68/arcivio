<script lang="ts">
export default { name: "SettingsTreeNode" };
</script>

<script setup lang="ts">
import { ref } from "vue";
import type { NavNode } from "./catalog";

const props = defineProps<{
  nodes: NavNode[];
  selectedId: string;
  depth?: number;
}>();

const emit = defineEmits<{
  select: [node: NavNode];
}>();

const expanded = ref<Record<string, boolean>>({});
const depth = props.depth ?? 0;

function isExpanded(id: string) {
  return expanded.value[id] !== false;
}

function toggle(id: string) {
  expanded.value[id] = !isExpanded(id);
}

function onClick(node: NavNode) {
  if (node.children?.length) {
    toggle(node.id);
  }
  emit("select", node);
}
</script>

<template>
  <li
    v-for="node in nodes"
    :key="node.id"
    role="treeitem"
    :aria-selected="node.id === selectedId"
    :aria-expanded="node.children?.length ? isExpanded(node.id) : undefined"
    :aria-level="depth + 1"
  >
    <button
      type="button"
      class="row"
      :class="{ selected: node.id === selectedId && !!node.item.page }"
      :style="{ paddingLeft: `${0.55 + depth * 0.85}rem` }"
      @click="onClick(node)"
    >
      <i
        v-if="node.children?.length"
        class="pi"
        :class="isExpanded(node.id) ? 'pi-chevron-down' : 'pi-chevron-right'"
        aria-hidden="true"
        @click.stop="toggle(node.id)"
      />
      <span>{{ node.label }}</span>
    </button>
    <ul v-if="node.children?.length && isExpanded(node.id)" role="group">
      <SettingsTreeNode :nodes="node.children" :selected-id="selectedId" :depth="depth + 1" @select="emit('select', $event)" />
    </ul>
  </li>
</template>

<style scoped>
li {
  list-style: none;
}
ul {
  margin: 0;
  padding: 0;
}
.row {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  width: 100%;
  margin: 0 0 0.15rem;
  padding: 0.42rem 0.55rem;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: #1b1f24;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.row:hover {
  background: #eef3fb;
}
.row.selected {
  background: #eef3fb;
  color: #1f4b99;
  font-weight: 600;
}
.row i {
  font-size: 0.7rem;
  color: #5b6570;
}
</style>
