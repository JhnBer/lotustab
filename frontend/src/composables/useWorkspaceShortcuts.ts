import { useEventListener } from "@vueuse/core";
import { useRoute, useRouter } from "vue-router";
import { matchWorkspaceShortcut } from "@/lib/shortcuts";
import { useProfileActions } from "@/composables/useProfileActions";
import { useConnectionsStore } from "@/stores/connections";
import { isTableTab, useQueryStore } from "@/stores/query";

export function useWorkspaceShortcuts() {
  const route = useRoute();
  const router = useRouter();
  const connections = useConnectionsStore();
  const queries = useQueryStore();
  const { openInNewWindow } = useProfileActions();

  function goQuery() {
    if (route.path !== "/query") {
      void router.push("/query");
    }
  }

  function cycleTab(delta: 1 | -1) {
    const profileId = connections.selectedId;
    if (!profileId) {
      return;
    }
    const group = queries.tabs.filter((t) => t.profileId === profileId);
    if (group.length === 0) {
      return;
    }
    const index = group.findIndex((t) => t.id === queries.activeTabId);
    const from = index < 0 ? 0 : index;
    const next = group[(from + delta + group.length) % group.length];
    if (!next) {
      return;
    }
    queries.selectTab(next.id);
    goQuery();
  }

  function newQueryTab() {
    const profile = connections.selected;
    if (!profile) {
      return;
    }
    queries.add(profile.id, profile.name);
    goQuery();
  }

  async function newWindow() {
    const profile = connections.selected;
    if (!profile) {
      if (route.path !== "/connections") {
        await router.push("/connections");
      }
      return;
    }
    await openInNewWindow(profile);
  }

  useEventListener("keydown", (event: KeyboardEvent) => {
    const id = matchWorkspaceShortcut(event);
    if (!id) {
      return;
    }
    event.preventDefault();
    switch (id) {
      case "tab.next":
        cycleTab(1);
        break;
      case "tab.prev":
        cycleTab(-1);
        break;
      case "tab.close":
        if (queries.activeTabId) {
          queries.close(queries.activeTabId);
        }
        break;
      case "tab.reload": {
        const tab = queries.activeTab;
        if (isTableTab(tab)) {
          void queries.reloadTab(tab.id);
        }
        break;
      }
      case "tab.new":
        newQueryTab();
        break;
      case "window.new":
        void newWindow();
        break;
    }
  });

  return { cycleTab, newQueryTab, newWindow };
}
