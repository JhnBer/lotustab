import { createPinia } from "pinia";
import { createApp } from "vue";
import "vue-sonner/style.css";
import App from "./App.vue";
import {
  bindTitleBarHeight,
  syncTitleBarHeight,
} from "./composables/useTitleBarHeight";
import { router } from "./router";
import "./style.css";

await syncTitleBarHeight();
bindTitleBarHeight();

createApp(App).use(createPinia()).use(router).mount("#app");
