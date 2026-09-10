<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from "vue";
import { ArrowUpDown, Database, Server, Settings } from "@lucide/vue";
import { storeToRefs } from "pinia";
import { VueDraggable } from "vue-draggable-plus";
import { useRoute, useRouter } from "vue-router";
import { toast } from "vue-sonner";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
  useSidebar,
} from "@/components/ui/sidebar";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuLabel,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import ConnectionDialog from "@/features/connections/ConnectionDialog.vue";
import { useConnectionDialog } from "@/composables/useConnectionDialog";
import { useConnectionTint } from "@/composables/useConnectionTint";
import { useProfileActions } from "@/composables/useProfileActions";
import { connBg } from "@/lib/colors";
import { errorMessage } from "@/lib/errors";
import type { Profile } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import { useTransfersStore } from "@/stores/transfers";

const route = useRoute();
const router = useRouter();
const connections = useConnectionsStore();
const transfers = useTransfersStore();
const { setOpen } = useSidebar();
const { pinnedProfiles } = storeToRefs(connections);
const { dialogOpen, editing, openEdit } = useConnectionDialog();
const { activateProfile, connect, disconnect, unpin, openInNewWindow } = useProfileActions();
const { tintClass: connectionsTint } = useConnectionTint();
const menuOpenId = ref("");
const sortedPinned = ref<Profile[]>([]);
const pinDragging = ref(false);

watch(
  pinnedProfiles,
  (next) => {
    if (pinDragging.value) {
      return;
    }
    sortedPinned.value = [...next];
  },
  { immediate: true },
);

function goConnections() {
  connections.select("");
  router.push("/connections");
  setOpen(true);
}

function goSettings() {
  if (route.path.startsWith("/settings")) {
    if (connections.selectedId) {
      router.push("/query");
    } else {
      router.push("/connections");
    }
    return;
  }
  router.push("/settings");
  setOpen(true);
}

function goTransfers() {
  if (route.path.startsWith("/transfers")) {
    if (connections.selectedId) {
      router.push("/query");
    } else {
      router.push("/connections");
    }
    return;
  }
  router.push("/transfers");
  setOpen(true);
}

function onPinDragStart() {
  pinDragging.value = true;
  window.addEventListener("pointerup", resetPinDragging, { once: true });
  window.addEventListener("pointercancel", resetPinDragging, { once: true });
}

async function onPinReorder() {
  try {
    await connections.reorderPinned(sortedPinned.value.map((p) => p.id));
  } catch (err) {
    sortedPinned.value = [...pinnedProfiles.value];
    handleErr(err);
  } finally {
    requestAnimationFrame(() => {
      pinDragging.value = false;
    });
  }
}

function resetPinDragging() {
  pinDragging.value = false;
}

function handleErr(err: unknown) {
  toast.error(errorMessage(err));
}

function onPinMenuOpen(id: string, open: boolean) {
  menuOpenId.value = open ? id : "";
}

function onActivateProfile(profile: Profile) {
  if (pinDragging.value) {
    return;
  }
  activateProfile(profile);
}

onBeforeUnmount(() => {
  window.removeEventListener("pointerup", resetPinDragging);
  window.removeEventListener("pointercancel", resetPinDragging);
});
</script>

<template>
  <Sidebar
    collapsible="none"
    class="h-full w-[calc(var(--sidebar-width-icon)+1px)]! shrink-0 border-r bg-background"
  >
    <SidebarContent class="relative">
      <SidebarGroup class="h-12" :class="connectionsTint">
        <SidebarGroupContent class="px-1.5 md:px-0">
          <SidebarMenu>
            <SidebarMenuItem>
              <Tooltip>
                <TooltipTrigger as-child>
                  <div class="w-full">
                    <SidebarMenuButton
                      :is-active="route.path === '/connections'"
                      class="px-2.5 md:px-2"
                      @click="goConnections"
                    >
                      <Server />
                      <span>Connections</span>
                    </SidebarMenuButton>
                  </div>
                </TooltipTrigger>
                <TooltipContent side="right" align="center">
                  Connections
                </TooltipContent>
              </Tooltip>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>

      <template v-if="pinnedProfiles.length > 0">
        <SidebarSeparator class="m-0 -translate-y-full" />
        <SidebarGroup>
          <SidebarGroupContent class="px-1.5 md:px-0">
            <VueDraggable
              v-model="sortedPinned"
              tag="div"
              :animation="150"
              :force-fallback="true"
              :fallback-on-body="true"
              :fallback-tolerance="4"
              class="flex w-full flex-col items-center gap-1"
              style="--wails-draggable: no-drag"
              @start="onPinDragStart"
              @end="onPinReorder"
            >
              <div
                v-for="profile in sortedPinned"
                :key="profile.id"
                class="size-8 select-none"
              >
                <ContextMenu @update:open="(open) => onPinMenuOpen(profile.id, open)">
                  <ContextMenuTrigger class="flex w-full justify-center">
                    <Tooltip :disabled="menuOpenId === profile.id || pinDragging">
                      <TooltipTrigger as-child>
                        <button
                          type="button"
                          class="flex size-8 cursor-grab items-center justify-center rounded-md text-foreground ring-offset-background focus-visible:ring-2 focus-visible:outline-none active:cursor-grabbing dark:text-white"
                          :class="connBg(profile.color) + (connections.selectedId === profile.id ? '/60' : '/20')"
                          @click="onActivateProfile(profile)"
                        >
                          <Database class="size-4" />
                          <span class="sr-only">{{ profile.name }}</span>
                        </button>
                      </TooltipTrigger>
                      <TooltipContent side="right" align="center">
                        {{ profile.name }}
                      </TooltipContent>
                    </Tooltip>
                  </ContextMenuTrigger>
                  <ContextMenuContent>
                    <ContextMenuLabel>{{ profile.name }}</ContextMenuLabel>
                    <ContextMenuSeparator />
                    <ContextMenuItem @click="openInNewWindow(profile)">
                      Open in new window
                    </ContextMenuItem>
                    <ContextMenuItem @click="unpin(profile)">
                      Unpin
                    </ContextMenuItem>
                    <ContextMenuItem @click="openEdit(profile)">
                      Edit
                    </ContextMenuItem>
                    <ContextMenuSeparator />
                    <template v-if="connections.isConnected(profile.id)">
                      <ContextMenuItem @click="disconnect(profile)">
                        Disconnect
                      </ContextMenuItem>
                    </template>
                    <template v-else-if="connections.isConnecting(profile.id)">
                      <ContextMenuItem @click="connections.cancelConnect(profile.id)">
                        Cancel connecting
                      </ContextMenuItem>
                    </template>
                    <template v-else>
                      <ContextMenuItem @click="connect(profile)">
                        Connect
                      </ContextMenuItem>
                    </template>
                  </ContextMenuContent>
                </ContextMenu>
              </div>
            </VueDraggable>
          </SidebarGroupContent>
        </SidebarGroup>
      </template>
    </SidebarContent>
    <SidebarFooter>
      <SidebarMenu>
        <SidebarMenuItem>
          <Tooltip>
            <TooltipTrigger as-child>
              <div class="w-full">
                <SidebarMenuButton
                  :is-active="route.path.startsWith('/transfers')"
                  class="relative px-2.5 md:px-2"
                  @click="goTransfers"
                >
                  <ArrowUpDown />
                  <span>Transfers</span>
                  <span
                    v-if="transfers.runningCount > 0"
                    class="bg-primary text-primary-foreground absolute top-0.5 right-0.5 flex size-3.5 items-center justify-center rounded-full text-[9px] font-medium"
                  >
                    {{ transfers.runningCount }}
                  </span>
                </SidebarMenuButton>
              </div>
            </TooltipTrigger>
            <TooltipContent side="right" align="center">
              Transfers
            </TooltipContent>
          </Tooltip>
        </SidebarMenuItem>
        <SidebarMenuItem>
          <Tooltip>
            <TooltipTrigger as-child>
              <div class="w-full">
                <SidebarMenuButton
                  :is-active="route.path.startsWith('/settings')"
                  class="px-2.5 md:px-2"
                  @click="goSettings"
                >
                  <Settings />
                  <span>Settings</span>
                </SidebarMenuButton>
              </div>
            </TooltipTrigger>
            <TooltipContent side="right" align="center">
              Settings
            </TooltipContent>
          </Tooltip>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarFooter>
    <ConnectionDialog v-model:open="dialogOpen" :profile="editing" />
  </Sidebar>
</template>
