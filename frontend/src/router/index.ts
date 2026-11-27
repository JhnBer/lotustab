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
      meta: { titleKey: "nav.connections" },
    },
    {
      path: "/query",
      name: "query",
      component: QueryWorkspace,
      meta: { titleKey: "nav.query" },
    },
    {
      path: "/value",
      name: "value",
      component: ValueInspectorPage,
      meta: { titleKey: "nav.value", chrome: "minimal" },
    },
    {
      path: "/transfers",
      name: "transfers",
      component: TransfersPage,
      meta: { titleKey: "nav.transfers" },
    },
    {
      path: "/settings",
      name: "settings",
      component: SettingsPage,
      meta: { titleKey: "nav.settings" },
    },
    {
      path: "/settings/shortcuts",
      name: "settings-shortcuts",
      component: ShortcutsPage,
      meta: { titleKey: "nav.settings" },
    },
    {
      path: "/settings/about",
      name: "settings-about",
      component: AboutPage,
      meta: { titleKey: "nav.settings" },
    },
  ],
});
