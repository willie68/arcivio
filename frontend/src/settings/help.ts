import { inject, onUnmounted, provide, ref, type Ref } from "vue";

const settingsHelpKey = Symbol("settingsHelp");

/** Page-level help text that replaces the catalog help while it is set. */
export function provideSettingsHelp(): Ref<string | null> {
  const text = ref<string | null>(null);
  provide(settingsHelpKey, text);
  return text;
}

/** Lets a settings page put its own text into the help pane. */
export function useSettingsHelp(): { set(text: string | null): void } {
  const text = inject<Ref<string | null> | null>(settingsHelpKey, null);
  function set(next: string | null) {
    if (text) {
      text.value = next;
    }
  }
  onUnmounted(() => set(null));
  return { set };
}
