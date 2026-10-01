<script setup lang="ts">
import { Pencil, Pin, Plug, Unplug } from "@lucide/vue";
import { storeToRefs } from "pinia";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { Spinner } from "@/components/ui/spinner";
import ConnectionDialog from "@/features/connections/ConnectionDialog.vue";
import { useConnectionDialog } from "@/composables/useConnectionDialog";
import { useExportData } from "@/composables/useExportData";
import { useProfileActions } from "@/composables/useProfileActions";
import { connBg } from "@/lib/colors";
import { useConnectionsStore } from "@/stores/connections";

const connections = useConnectionsStore();
const { recentProfiles } = storeToRefs(connections);
const { dialogOpen, editing, openEdit } = useConnectionDialog();
const { openExportData } = useExportData();
const { activateProfile, togglePin, connect, disconnect, clone, remove } = useProfileActions();
</script>

<template>
  <SidebarGroup class="min-h-0 flex-1 overflow-hidden">
    <SidebarGroupLabel class="bg-sidebar shrink-0">Recent</SidebarGroupLabel>
    <SidebarGroupContent class="min-h-0 flex-1 overflow-auto">
      <p v-if="recentProfiles.length === 0" class="text-muted-foreground px-2 py-2 text-xs">
        No recent connections.
      </p>
      <SidebarMenu v-else>
        <SidebarMenuItem v-for="profile in recentProfiles" :key="profile.id">
          <ContextMenu>
            <ContextMenuTrigger as-child>
              <div class="flex items-center gap-0.5">
                <SidebarMenuButton class="min-w-0 flex-1" @click="activateProfile(profile)">
                  <span
                    class="size-2 shrink-0 rounded-full"
                    :class="connBg(profile.color)"
                  />
                  <span class="truncate">{{ profile.name }}</span>
                </SidebarMenuButton>
                <Button
                  size="icon-xs"
                  variant="ghost"
                  class="shrink-0"
                  :title="connections.isPinned(profile.id) ? 'Unpin' : 'Pin'"
                  @click.stop="togglePin(profile)"
                >
                  <Pin
                    class="size-3.5"
                    :class="connections.isPinned(profile.id) ? 'fill-current' : ''"
                  />
                </Button>
                <Button
                  v-if="connections.isConnecting(profile.id)"
                  size="icon-xs"
                  variant="ghost"
                  class="shrink-0"
                  title="Cancel"
                  @click.stop="connections.cancelConnect(profile.id)"
                >
                  <Spinner class="size-3.5" />
                </Button>
                <Button
                  v-else-if="!connections.isConnected(profile.id)"
                  size="icon-xs"
                  variant="ghost"
                  class="shrink-0"
                  title="Connect"
                  @click.stop="connect(profile)"
                >
                  <Plug class="size-3.5" />
                </Button>
                <Button
                  v-else
                  size="icon-xs"
                  variant="ghost"
                  class="shrink-0"
                  title="Disconnect"
                  @click.stop="disconnect(profile)"
                >
                  <Unplug class="size-3.5" />
                </Button>
                <Button
                  size="icon-xs"
                  variant="ghost"
                  class="shrink-0"
                  title="Edit"
                  @click.stop="openEdit(profile)"
                >
                  <Pencil class="size-3.5" />
                </Button>
              </div>
            </ContextMenuTrigger>
            <ContextMenuContent>
              <ContextMenuItem
                v-if="connections.isConnected(profile.id)"
                @select="disconnect(profile)"
              >
                Disconnect
              </ContextMenuItem>
              <ContextMenuItem
                v-else-if="connections.isConnecting(profile.id)"
                @select="connections.cancelConnect(profile.id)"
              >
                Cancel connecting
              </ContextMenuItem>
              <ContextMenuItem v-else @select="connect(profile)">
                Connect
              </ContextMenuItem>
              <ContextMenuItem @select="togglePin(profile)">
                {{ connections.isPinned(profile.id) ? "Unpin" : "Pin" }}
              </ContextMenuItem>
              <ContextMenuItem @select="openEdit(profile)">
                Edit
              </ContextMenuItem>
              <ContextMenuItem @select="openExportData(profile.id)">
                Export…
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
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarGroupContent>
  </SidebarGroup>

  <ConnectionDialog v-model:open="dialogOpen" :profile="editing" />
</template>
