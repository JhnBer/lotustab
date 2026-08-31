import { defineStore } from "pinia";
import { computed, ref, watch } from "vue";
import { Events } from "@wailsio/runtime";
import { errorMessage } from "@/lib/errors";
import { readJson, writeJson } from "@/lib/persist";
import {
  ConnectionsService,
  DriversService,
  WindowsService,
  type DriverInfo,
  type Field,
  type Profile,
  type SaveProfileInput,
} from "@/lib/wails";
import { useExplorerStore } from "@/stores/explorer";
import { useQueryStore } from "@/stores/query";

const SELECTED_KEY = "lotustab.selectedId";

function asRecord(params: Profile["params"]): Record<string, string> {
  const out: Record<string, string> = {};
  if (!params) {
    return out;
  }
  for (const [key, value] of Object.entries(params)) {
    if (typeof value === "string") {
      out[key] = value;
    }
  }
  return out;
}

export const useConnectionsStore = defineStore("connections", () => {
  const profiles = ref<Profile[]>([]);
  const drivers = ref<DriverInfo[]>([]);
  const connectedIds = ref<string[]>([]);
  const pinnedIds = ref<string[]>([]);
  const recentIds = ref<string[]>([]);
  const storedSelected = readJson<unknown>(SELECTED_KEY, "");
  const selectedId = ref(typeof storedSelected === "string" ? storedSelected : "");
  const connectingIds = ref<string[]>([]);
  const pendingConnect = new Map<string, Promise<void>>();
  const loading = ref(false);
  const error = ref("");

  watch(selectedId, (id) => writeJson(SELECTED_KEY, id));

  const selected = computed(() => profiles.value.find((p) => p.id === selectedId.value));

  const pinnedProfiles = computed(() =>
    pinnedIds.value
      .map((id) => profiles.value.find((p) => p.id === id))
      .filter((p): p is Profile => Boolean(p)),
  );

  const recentProfiles = computed(() =>
    recentIds.value
      .map((id) => profiles.value.find((p) => p.id === id))
      .filter((p): p is Profile => Boolean(p)),
  );

  function isConnected(id: string) {
    return connectedIds.value.includes(id);
  }

  function isConnecting(id: string) {
    return connectingIds.value.includes(id);
  }

  function isPinned(id: string) {
    return pinnedIds.value.includes(id);
  }

  function driverById(id: string) {
    return drivers.value.find((d) => d.id === id);
  }

  function specFor(driverId: string): Field[] {
    return driverById(driverId)?.connectionSpec ?? [];
  }

  function applyState(state: {
    profiles?: Profile[] | null;
    pinnedIds?: string[] | null;
    recentIds?: string[] | null;
  }) {
    profiles.value = state.profiles ?? [];
    pinnedIds.value = state.pinnedIds ?? [];
    recentIds.value = state.recentIds ?? [];
    if (selectedId.value && !profiles.value.some((p) => p.id === selectedId.value)) {
      selectedId.value = "";
    }
  }

  function setConnectedIds(next: string[] | null | undefined) {
    const ids = next ?? [];
    const prev = connectedIds.value;
    connectedIds.value = ids;
    const still = new Set(ids);
    for (const id of prev) {
      if (!still.has(id)) {
        useQueryStore().dropTableData(id);
      }
    }
  }

  async function refresh() {
    loading.value = true;
    error.value = "";
    try {
      const [state, nextDrivers, nextConnected] = await Promise.all([
        ConnectionsService.List(),
        DriversService.List(),
        ConnectionsService.ConnectedIDs(),
      ]);
      applyState(state);
      drivers.value = nextDrivers ?? [];
      setConnectedIds(nextConnected);
    } catch (err) {
      error.value = errorMessage(err);
      throw err;
    } finally {
      loading.value = false;
    }
  }

  async function refreshConnected() {
    setConnectedIds(await ConnectionsService.ConnectedIDs());
  }

  async function refreshNav() {
    applyState(await ConnectionsService.List());
  }

  async function syncFromBackend() {
    try {
      const [state, nextConnected] = await Promise.all([
        ConnectionsService.List(),
        ConnectionsService.ConnectedIDs(),
      ]);
      applyState(state);
      setConnectedIds(nextConnected);
    } catch {
      // Keep the current window usable if a sync tick fails.
    }
  }

  Events.On("connections:changed", () => {
    void syncFromBackend();
  });

  async function save(input: SaveProfileInput) {
    const saved = await ConnectionsService.Save({
      ...input,
      params: input.params ?? {},
    });
    await refresh();
    return saved;
  }

  async function test(input: SaveProfileInput) {
    await ConnectionsService.Test({
      ...input,
      params: input.params ?? {},
    });
  }

  async function fetchDatabases(input: SaveProfileInput) {
    return (
      (await ConnectionsService.FetchDatabases({
        ...input,
        params: input.params ?? {},
      })) ?? []
    );
  }

  async function remove(id: string) {
    await ConnectionsService.Delete(id);
    useQueryStore().removeForProfile(id);
    useExplorerStore().removeProfile(id);
    await refresh();
  }

  async function connect(id: string) {
    if (connectedIds.value.includes(id)) {
      return;
    }
    const pending = pendingConnect.get(id);
    if (pending) {
      return pending;
    }
    connectingIds.value = [...connectingIds.value, id];
    const work = (async () => {
      try {
        await ConnectionsService.Connect(id);
        await refresh();
      } finally {
        connectingIds.value = connectingIds.value.filter((item) => item !== id);
        pendingConnect.delete(id);
      }
    })();
    pendingConnect.set(id, work);
    return work;
  }

  async function disconnect(id: string) {
    await ConnectionsService.Disconnect(id);
    await refreshConnected();
  }

  function select(id: string) {
    selectedId.value = id;
    void touchRecent(id);
  }

  async function openInNewWindow(profile: Profile) {
    const queries = useQueryStore();
    const snap = queries.snapshotForProfile(profile.id);
    await WindowsService.OpenConnection(
      profile.id,
      profile.name,
      snap.tabs.length > 0 ? JSON.stringify(snap) : "",
    );
    queries.removeForProfile(profile.id);
    if (selectedId.value === profile.id) {
      selectedId.value = "";
    }
  }

  async function setPinned(id: string, pinned: boolean) {
    await ConnectionsService.SetPinned(id, pinned);
    await refreshNav();
  }

  async function setColor(id: string, color: string) {
    await ConnectionsService.SetColor(id, color);
    await refreshNav();
  }

  async function touchRecent(id: string) {
    if (!id) {
      return;
    }
    try {
      await ConnectionsService.TouchRecent(id);
      await refreshNav();
    } catch {
      // Selection still works if prefs fail to persist.
    }
  }

  return {
    profiles,
    drivers,
    connectedIds,
    pinnedIds,
    recentIds,
    selectedId,
    connectingIds,
    loading,
    error,
    selected,
    pinnedProfiles,
    recentProfiles,
    isConnected,
    isConnecting,
    isPinned,
    driverById,
    specFor,
    asRecord,
    refresh,
    save,
    test,
    fetchDatabases,
    remove,
    connect,
    disconnect,
    select,
    openInNewWindow,
    setPinned,
    setColor,
    touchRecent,
  };
});
