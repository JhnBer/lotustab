import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { errorMessage } from "@/lib/errors";
import {
  ConnectionsService,
  DriversService,
  type DriverInfo,
  type Field,
  type Profile,
  type SaveProfileInput,
} from "@/lib/wails";

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
  const selectedId = ref("");
  const loading = ref(false);
  const error = ref("");

  const selected = computed(() => profiles.value.find((p) => p.id === selectedId.value));

  const pinnedProfiles = computed(() =>
    pinnedIds.value
      .map((id) => profiles.value.find((p) => p.id === id))
      .filter((p): p is Profile => Boolean(p)),
  );

  const recentProfiles = computed(() =>
    recentIds.value
      .filter((id) => id !== selectedId.value)
      .map((id) => profiles.value.find((p) => p.id === id))
      .filter((p): p is Profile => Boolean(p)),
  );

  function isConnected(id: string) {
    return connectedIds.value.includes(id);
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
      selectedId.value = profiles.value[0]?.id ?? "";
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
      connectedIds.value = nextConnected ?? [];
    } catch (err) {
      error.value = errorMessage(err);
      throw err;
    } finally {
      loading.value = false;
    }
  }

  async function refreshConnected() {
    connectedIds.value = (await ConnectionsService.ConnectedIDs()) ?? [];
  }

  async function refreshNav() {
    applyState(await ConnectionsService.List());
  }

  async function save(input: SaveProfileInput) {
    const saved = await ConnectionsService.Save({
      ...input,
      params: input.params ?? {},
    });
    await refresh();
    selectedId.value = saved.id;
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
    await refresh();
  }

  async function connect(id: string) {
    await ConnectionsService.Connect(id);
    selectedId.value = id;
    await refresh();
  }

  async function disconnect(id: string) {
    await ConnectionsService.Disconnect(id);
    await refreshConnected();
  }

  function select(id: string) {
    selectedId.value = id;
    void touchRecent(id);
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
    loading,
    error,
    selected,
    pinnedProfiles,
    recentProfiles,
    isConnected,
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
    setPinned,
    setColor,
    touchRecent,
  };
});
