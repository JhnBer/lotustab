<script setup lang="ts">
import { ref } from "vue";
import { Pencil, Pin, Plug, Plus, Trash2, Unplug } from "@lucide/vue";
import { useRouter } from "vue-router";
import { toast } from "vue-sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import ConnectionDialog from "@/features/connections/ConnectionDialog.vue";
import { errorMessage } from "@/lib/errors";
import { connBg } from "@/lib/colors";
import type { Profile } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import { useQueryStore } from "@/stores/query";

const router = useRouter();
const connections = useConnectionsStore();
const queries = useQueryStore();

const dialogOpen = ref(false);
const editing = ref<Profile | null>(null);

function param(profile: Profile, key: string) {
  return connections.asRecord(profile.params)[key] || "—";
}

function openCreate() {
  editing.value = null;
  dialogOpen.value = true;
}

function openEdit(profile: Profile) {
  editing.value = profile;
  dialogOpen.value = true;
}

function openQuery(profile: Profile) {
  connections.select(profile.id);
  queries.openForProfile(profile.id, profile.name);
  router.push("/query");
}

async function connect(profile: Profile) {
  try {
    await connections.connect(profile.id);
    connections.select(profile.id);
    queries.openForProfile(profile.id, profile.name);
    toast.success(`Connected to ${profile.name}`);
    router.push("/query");
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

async function disconnect(profile: Profile) {
  try {
    await connections.disconnect(profile.id);
    toast.success(`Disconnected ${profile.name}`);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

async function togglePin(profile: Profile) {
  try {
    await connections.setPinned(profile.id, !connections.isPinned(profile.id));
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

async function remove(profile: Profile) {
  try {
    await connections.remove(profile.id);
    toast.success(`Deleted ${profile.name}`);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col gap-4">
    <div class="flex items-center justify-between gap-2">
      <div>
        <h1 class="text-lg font-semibold tracking-tight">Connections</h1>
        <p class="text-muted-foreground text-sm">
          Saved profiles. Passwords stay in the OS keychain.
        </p>
      </div>
      <Button size="sm" @click="openCreate">
        <Plus class="size-3.5" />
        New connection
      </Button>
    </div>

    <div class="min-h-0 flex-1 overflow-auto border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Driver</TableHead>
            <TableHead>Host</TableHead>
            <TableHead>Port</TableHead>
            <TableHead>Database</TableHead>
            <TableHead>Status</TableHead>
            <TableHead class="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableEmpty v-if="connections.profiles.length === 0" :colspan="7">
            <div class="text-muted-foreground flex flex-col items-center gap-2">
              <span>No connections yet.</span>
              <Button size="sm" variant="outline" @click="openCreate">
                Add PostgreSQL or MySQL
              </Button>
            </div>
          </TableEmpty>
          <TableRow v-for="profile in connections.profiles" :key="profile.id">
            <TableCell>
              <button
                type="button"
                class="flex items-center gap-2 font-medium hover:underline"
                @click="openQuery(profile)"
              >
                <span class="size-2.5 shrink-0 rounded-full" :class="connBg(profile.color)" />
                {{ profile.name }}
              </button>
            </TableCell>
            <TableCell>
              <Badge variant="secondary">
                {{ connections.driverById(profile.driverId)?.displayName ?? profile.driverId }}
              </Badge>
            </TableCell>
            <TableCell class="font-mono text-xs">{{ param(profile, "host") }}</TableCell>
            <TableCell class="font-mono text-xs">{{ param(profile, "port") }}</TableCell>
            <TableCell class="font-mono text-xs">{{ param(profile, "database") }}</TableCell>
            <TableCell>
              <Badge :variant="connections.isConnected(profile.id) ? 'default' : 'secondary'">
                {{ connections.isConnected(profile.id) ? "Connected" : "Offline" }}
              </Badge>
            </TableCell>
            <TableCell>
              <div class="flex justify-end gap-1">
                <Button
                  size="icon-sm"
                  variant="ghost"
                  :title="connections.isPinned(profile.id) ? 'Unpin' : 'Pin'"
                  @click="togglePin(profile)"
                >
                  <Pin
                    class="size-3.5"
                    :class="connections.isPinned(profile.id) ? 'fill-current' : ''"
                  />
                  <span class="sr-only">{{ connections.isPinned(profile.id) ? "Unpin" : "Pin" }}</span>
                </Button>
                <Button
                  v-if="!connections.isConnected(profile.id)"
                  size="icon-sm"
                  variant="ghost"
                  @click="connect(profile)"
                >
                  <Plug class="size-3.5" />
                  <span class="sr-only">Connect</span>
                </Button>
                <Button
                  v-else
                  size="icon-sm"
                  variant="ghost"
                  @click="disconnect(profile)"
                >
                  <Unplug class="size-3.5" />
                  <span class="sr-only">Disconnect</span>
                </Button>
                <Button size="icon-sm" variant="ghost" @click="openEdit(profile)">
                  <Pencil class="size-3.5" />
                  <span class="sr-only">Edit</span>
                </Button>
                <Button size="icon-sm" variant="ghost" @click="remove(profile)">
                  <Trash2 class="size-3.5" />
                  <span class="sr-only">Delete</span>
                </Button>
              </div>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <ConnectionDialog v-model:open="dialogOpen" :profile="editing" />
  </div>
</template>
