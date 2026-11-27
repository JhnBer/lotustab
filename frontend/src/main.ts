import { createPinia } from "pinia";
import { watch } from "vue";
import { createApp } from "vue";
import "vue-sonner/style.css";
import App from "./App.vue";
import {
  bindTitleBarHeight,
  syncTitleBarHeight,
} from "./composables/useTitleBarHeight";
import { i18n } from "./i18n";
import { router } from "./router";
import { useSettingsStore } from "./stores/settings";
import "./style.css";

await syncTitleBarHeight();
bindTitleBarHeight();

const app = createApp(App);
const pinia = createPinia();
app.use(pinia);
app.use(i18n);
app.use(router);

const settings = useSettingsStore();
watch(
  () => settings.locale,
  (locale) => {
    i18n.global.locale.value = locale;
    document.documentElement.lang = locale;
  },
  { immediate: true },
);

app.mount("#app");
