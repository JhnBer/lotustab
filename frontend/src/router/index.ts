import { createRouter, createWebHashHistory } from "vue-router";
import ConnectionsPage from "@/features/connections/ConnectionsPage.vue";
import QueryWorkspace from "@/features/editor/QueryWorkspace.vue";
import ValueInspectorPage from "@/features/results/ValueInspectorPage.vue";
import SettingsPage from "@/features/settings/SettingsPage.vue";
import ShortcutsPage from "@/features/settings/ShortcutsPage.vue";
import AboutPage from "@/features/settings/AboutPage.vue";
import TransfersPage from "@/features/transfers/TransfersPage.vue";

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
      path: "/value",
      name: "value",
      component: ValueInspectorPage,
      meta: { title: "Value", chrome: "minimal" },
    },
    {
      path: "/transfers",
      name: "transfers",
      component: TransfersPage,
      meta: { title: "Transfers" },
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
      meta: { title: "Settings" },
    },
    {
      path: "/settings/about",
      name: "settings-about",
      component: AboutPage,
      meta: { title: "Settings" },
    },
  ],
});
