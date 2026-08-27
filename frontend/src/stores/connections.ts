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
  const selectedId = ref("");
  const loading = ref(false);
  const error = ref("");

  const selected = computed(() => profiles.value.find((p) => p.id === selectedId.value));

  function isConnected(id: string) {
    return connectedIds.value.includes(id);
  }

  function driverById(id: string) {
    return drivers.value.find((d) => d.id === id);
  }

  function specFor(driverId: string): Field[] {
    return driverById(driverId)?.connectionSpec ?? [];
  }

  async function refresh() {
    loading.value = true;
    error.value = "";
    try {
      const [nextProfiles, nextDrivers, nextConnected] = await Promise.all([
        ConnectionsService.List(),
        DriversService.List(),
        ConnectionsService.ConnectedIDs(),
      ]);
      profiles.value = nextProfiles ?? [];
      drivers.value = nextDrivers ?? [];
      connectedIds.value = nextConnected ?? [];
      if (selectedId.value && !profiles.value.some((p) => p.id === selectedId.value)) {
        selectedId.value = profiles.value[0]?.id ?? "";
      }
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

  async function remove(id: string) {
    await ConnectionsService.Delete(id);
    await refresh();
  }

  async function connect(id: string) {
    await ConnectionsService.Connect(id);
    selectedId.value = id;
    await refreshConnected();
  }

  async function disconnect(id: string) {
    await ConnectionsService.Disconnect(id);
    await refreshConnected();
  }

  function select(id: string) {
    selectedId.value = id;
  }

  return {
    profiles,
    drivers,
    connectedIds,
    selectedId,
    loading,
    error,
    selected,
    isConnected,
    driverById,
    specFor,
    asRecord,
    refresh,
    save,
    test,
    remove,
    connect,
    disconnect,
    select,
  };
});
