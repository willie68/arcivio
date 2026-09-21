import { createApp, watch } from "vue";
import PrimeVue from "primevue/config";
import Aura from "@primeuix/themes/aura";
import Tooltip from "primevue/tooltip";
import "primeicons/primeicons.css";
import App from "./App.vue";
import router from "./router";
import { i18n, syncDocumentLang } from "./i18n";

const app = createApp(App);
app.use(i18n);
app.use(PrimeVue, {
  theme: {
    preset: Aura,
    options: {
      darkModeSelector: false,
    },
  },
});
app.directive("tooltip", Tooltip);
app.use(router);

syncDocumentLang(String(i18n.global.locale.value));
watch(i18n.global.locale, (locale) => {
  syncDocumentLang(locale);
});

app.mount("#app");
