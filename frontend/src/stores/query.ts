import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { errorMessage } from "@/lib/errors";
import { QueryService, type QueryResult } from "@/lib/wails";

export interface QueryTab {
  id: string;
  title: string;
  profileId: string;
  sql: string;
  running: boolean;
  error: string;
  result: QueryResult | null;
}

function newTab(profileId: string, title: string): QueryTab {
  return {
    id: crypto.randomUUID(),
    title,
    profileId,
    sql: "SELECT 1;",
    running: false,
    error: "",
    result: null,
  };
}

export const useQueryStore = defineStore("query", () => {
  const tabs = ref<QueryTab[]>([]);
  const activeTabId = ref("");

  const activeTab = computed(() => tabs.value.find((t) => t.id === activeTabId.value));

  function openForProfile(profileId: string, name: string) {
    const existing = tabs.value.find((t) => t.profileId === profileId);
    if (existing) {
      activeTabId.value = existing.id;
      return existing;
    }
    const tab = newTab(profileId, name);
    tabs.value.push(tab);
    activeTabId.value = tab.id;
    return tab;
  }

  function add(profileId: string, name: string) {
    const count = tabs.value.filter((t) => t.profileId === profileId).length;
    const tab = newTab(profileId, count === 0 ? name : `${name} ${count + 1}`);
    tabs.value.push(tab);
    activeTabId.value = tab.id;
    return tab;
  }

  function close(id: string) {
    const index = tabs.value.findIndex((t) => t.id === id);
    if (index < 0) {
      return;
    }
    tabs.value.splice(index, 1);
    if (activeTabId.value === id) {
      const next = tabs.value[index] ?? tabs.value[index - 1];
      activeTabId.value = next?.id ?? "";
    }
  }

  function setSql(id: string, sql: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (tab) {
      tab.sql = sql;
    }
  }

  async function run(id: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab) {
      return;
    }
    if (!tab.sql.trim()) {
      tab.error = "Query is empty";
      return;
    }
    tab.running = true;
    tab.error = "";
    try {
      tab.result = await QueryService.Execute(tab.profileId, tab.sql);
    } catch (err) {
      tab.result = null;
      tab.error = errorMessage(err);
    } finally {
      tab.running = false;
    }
  }

  return {
    tabs,
    activeTabId,
    activeTab,
    openForProfile,
    add,
    close,
    setSql,
    run,
  };
});
