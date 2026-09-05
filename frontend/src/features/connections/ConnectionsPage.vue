<script setup lang="ts">
import { computed } from "vue";
import { GripVertical, Pencil, Pin, Plug, Plus, Trash2, Unplug } from "@lucide/vue";
import { VueDraggable } from "vue-draggable-plus";
import { toast } from "vue-sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { Spinner } from "@/components/ui/spinner";
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
import { useConnectionDialog } from "@/composables/useConnectionDialog";
import { useProfileActions } from "@/composables/useProfileActions";
import { errorMessage } from "@/lib/errors";
import { connBg } from "@/lib/colors";
import { asRecord } from "@/lib/utils";
import type { Profile } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";

const connections = useConnectionsStore();
const { dialogOpen, editing, openCreate, openEdit } = useConnectionDialog();
const { activateProfile, connect, disconnect, togglePin, clone, remove } = useProfileActions();

const sortableProfiles = computed({
  get: () => connections.profiles,
  set: (next: Profile[]) => {
    connections.profiles = next;
  },
});

function param(profile: Profile, key: string) {
  return asRecord(profile.params)[key] || "—";
}

function openQuery(profile: Profile) {
  activateProfile(profile, { openTab: true });
}

async function connectAndOpen(profile: Profile) {
  await connect(profile, { openQuery: true });
}

async function onReorder() {
  try {
    await connections.reorderProfiles(sortableProfiles.value.map((p) => p.id));
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
            <TableHead class="w-8" />
            <TableHead>Name</TableHead>
            <TableHead>Driver</TableHead>
            <TableHead>Host</TableHead>
            <TableHead>Port</TableHead>
            <TableHead>Database</TableHead>
            <TableHead>Status</TableHead>
            <TableHead class="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody v-if="connections.profiles.length === 0">
          <TableEmpty :colspan="8">
            <div class="text-muted-foreground flex flex-col items-center gap-2">
              <span>No connections yet.</span>
              <Button size="sm" variant="outline" @click="openCreate">
                Add PostgreSQL, MySQL, or ClickHouse
              </Button>
            </div>
          </TableEmpty>
        </TableBody>
        <VueDraggable
          v-else
          v-model="sortableProfiles"
          tag="tbody"
          :animation="150"
          handle=".drag-handle"
          :force-fallback="true"
          class="[&_tr:last-child]:border-0"
          @end="onReorder"
        >
          <ContextMenu v-for="profile in sortableProfiles" :key="profile.id">
            <ContextMenuTrigger as-child>
              <TableRow>
                <TableCell class="w-8 px-1">
                  <button
                    type="button"
                    class="drag-handle text-muted-foreground hover:text-foreground flex cursor-grab items-center justify-center active:cursor-grabbing"
                    title="Reorder"
                    @click.stop
                  >
                    <GripVertical class="size-3.5" />
                    <span class="sr-only">Reorder</span>
                  </button>
                </TableCell>
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
                      v-if="connections.isConnecting(profile.id)"
                      size="icon-sm"
                      variant="ghost"
                      title="Cancel"
                      @click="connections.cancelConnect(profile.id)"
                    >
                      <Spinner class="size-3.5" />
                      <span class="sr-only">Cancel</span>
                    </Button>
                    <Button
                      v-else-if="!connections.isConnected(profile.id)"
                      size="icon-sm"
                      variant="ghost"
                      title="Connect"
                      @click="connectAndOpen(profile)"
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
            </ContextMenuTrigger>
            <ContextMenuContent>
              <ContextMenuItem v-if="connections.isConnected(profile.id)" @select="disconnect(profile)">
                Disconnect
              </ContextMenuItem>
              <ContextMenuItem
                v-else-if="connections.isConnecting(profile.id)"
                @select="connections.cancelConnect(profile.id)"
              >
                Cancel connecting
              </ContextMenuItem>
              <ContextMenuItem v-else @select="connectAndOpen(profile)">
                Connect
              </ContextMenuItem>
              <ContextMenuItem @select="togglePin(profile)">
                {{ connections.isPinned(profile.id) ? "Unpin" : "Pin" }}
              </ContextMenuItem>
              <ContextMenuItem @select="openEdit(profile)">
                Edit
              </ContextMenuItem>
              <ContextMenuItem @select="clone(profile)">
                Clone
              </ContextMenuItem>
              <ContextMenuSeparator />
              <ContextMenuItem variant="destructive" @select="remove(profile)">
                Delete
              </ContextMenuItem>
            </ContextMenuContent>
          </ContextMenu>
        </VueDraggable>
      </Table>
    </div>

    <ConnectionDialog v-model:open="dialogOpen" :profile="editing" />
  </div>
</template>
