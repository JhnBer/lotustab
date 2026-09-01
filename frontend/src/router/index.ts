import { createRouter, createWebHashHistory } from "vue-router";
import ConnectionsPage from "@/features/connections/ConnectionsPage.vue";
import QueryWorkspace from "@/features/editor/QueryWorkspace.vue";
import SettingsPage from "@/features/settings/SettingsPage.vue";
import ShortcutsPage from "@/features/settings/ShortcutsPage.vue";

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: "/", redirect: "/connections" },
    {
      path: "/connections",
      name: "connections",
      component: ConnectionsPage,
      meta: { title: "Connections" },
    },
    {
      path: "/query",
      name: "query",
      component: QueryWorkspace,
      meta: { title: "Query" },
    },
    {
      path: "/settings",
      name: "settings",
      component: SettingsPage,
      meta: { title: "Settings" },
    },
    {
      path: "/settings/shortcuts",
      name: "settings-shortcuts",
      component: ShortcutsPage,
      meta: { title: "Shortcuts" },
    },
  ],
});
