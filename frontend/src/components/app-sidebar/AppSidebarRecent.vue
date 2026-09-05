<script setup lang="ts">
import { Pencil, Pin, Plug, Unplug } from "@lucide/vue";
import { toast } from "vue-sonner";
import { storeToRefs } from "pinia";
import { ref } from "vue";
import { useRouter } from "vue-router";
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
import { errorMessage, isCanceled } from "@/lib/errors";
import { connBg } from "@/lib/colors";
import type { Profile } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";

const router = useRouter();
const connections = useConnectionsStore();
const { recentProfiles } = storeToRefs(connections);
const dialogOpen = ref(false);
const editing = ref<Profile | null>(null);

function activateProfile(profile: Profile) {
  connections.select(profile.id);
  router.push("/query");
}

async function togglePin(profile: Profile) {
  try {
    await connections.setPinned(profile.id, !connections.isPinned(profile.id));
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

async function connectRecent(profile: Profile) {
  if (connections.isConnecting(profile.id)) {
    connections.cancelConnect(profile.id);
    return;
  }
  try {
    await connections.connect(profile.id);
    if (!connections.isConnected(profile.id)) {
      return;
    }
    toast.success(`Connected to ${profile.name}`);
  } catch (err) {
    if (isCanceled(err)) {
      return;
    }
    toast.error(errorMessage(err));
  }
}

async function disconnectRecent(profile: Profile) {
  try {
    await connections.disconnect(profile.id);
    toast.info(`Disconnected from ${profile.name}`);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

function openEdit(profile: Profile) {
  editing.value = profile;
  dialogOpen.value = true;
}

async function cloneRecent(profile: Profile) {
  try {
    await connections.clone(profile.id);
    toast.success(`Cloned ${profile.name}`);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

async function removeRecent(profile: Profile) {
  try {
    await connections.remove(profile.id);
    toast.success(`Deleted ${profile.name}`);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}
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
                  @click.stop="connectRecent(profile)"
                >
                  <Plug class="size-3.5" />
                </Button>
                <Button
                  v-else
                  size="icon-xs"
                  variant="ghost"
                  class="shrink-0"
                  title="Disconnect"
                  @click.stop="disconnectRecent(profile)"
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
                @select="disconnectRecent(profile)"
              >
                Disconnect
              </ContextMenuItem>
              <ContextMenuItem
                v-else-if="connections.isConnecting(profile.id)"
                @select="connections.cancelConnect(profile.id)"
              >
                Cancel connecting
              </ContextMenuItem>
              <ContextMenuItem v-else @select="connectRecent(profile)">
                Connect
              </ContextMenuItem>
              <ContextMenuItem @select="togglePin(profile)">
                {{ connections.isPinned(profile.id) ? "Unpin" : "Pin" }}
              </ContextMenuItem>
              <ContextMenuItem @select="openEdit(profile)">
                Edit
              </ContextMenuItem>
              <ContextMenuItem @select="cloneRecent(profile)">
                Clone
              </ContextMenuItem>
              <ContextMenuSeparator />
              <ContextMenuItem variant="destructive" @select="removeRecent(profile)">
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
