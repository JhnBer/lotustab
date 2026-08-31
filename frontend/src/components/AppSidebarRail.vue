<script setup lang="ts">
import { computed } from "vue";
import { Database, Server, Settings } from "@lucide/vue";
import { storeToRefs } from "pinia";
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
import { Spinner } from "@/components/ui/spinner";
import { connBg, connTint } from "@/lib/colors";
import { errorMessage } from "@/lib/errors";
import type { Profile } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import { useSettingsStore } from "@/stores/settings";

const route = useRoute();
const router = useRouter();
const connections = useConnectionsStore();
const settings = useSettingsStore();
const { setOpen } = useSidebar();
const { pinnedProfiles } = storeToRefs(connections);

const connectionsTint = computed(() =>
  settings.connectionTint && connections.selected && !route.path.startsWith("/settings")
    ? connTint(connections.selected.color)
    : "",
);

function goConnections() {
  connections.select("");
  router.push("/connections");
  setOpen(true);
}

function goSettings() {
  router.push("/settings");
  setOpen(true);
}

function activateProfile(profile: Profile) {
  connections.select(profile.id);
  router.push("/query");
}

async function openInNewWindow(profile: Profile) {
  try {
    await connections.openInNewWindow(profile);
    if (!connections.selectedId) {
      await router.push("/connections");
    }
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

async function unpin(profile: Profile) {
  try {
    await connections.setPinned(profile.id, false);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

async function disconnect(profile: Profile) {
  try {
    await connections.disconnect(profile.id);
    toast.info(`Disconnected from ${profile.name}`);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

async function connect(profile: Profile) {
  if (connections.isConnecting(profile.id)) {
    return;
  }
  try {
    await connections.connect(profile.id);
    toast.success(`Connected to ${profile.name}`);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}
</script>

<template>
  <Sidebar
    collapsible="none"
    class="h-full w-[calc(var(--sidebar-width-icon)+1px)]! shrink-0 border-r"
  >
    <SidebarContent class="relative">
      <SidebarGroup class="h-12" :class="connectionsTint">
        <SidebarGroupContent class="px-1.5 md:px-0">
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton
                tooltip="Connections"
                :is-active="route.path === '/connections'"
                class="px-2.5 md:px-2"
                @click="goConnections"
              >
                <Server />
                <span>Connections</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>

      <template v-if="pinnedProfiles.length > 0">
        <SidebarSeparator class="m-0 -translate-y-full" />
        <SidebarGroup>
          <SidebarGroupContent class="px-1.5 md:px-0">
            <SidebarMenu class="gap-1">
              <SidebarMenuItem v-for="profile in pinnedProfiles" :key="profile.id">
                <ContextMenu>
                  <ContextMenuTrigger as-child>
                    <button
                      type="button"
                      class="mx-auto flex size-8 items-center justify-center rounded-md text-foreground ring-offset-background focus-visible:ring-2 focus-visible:outline-none dark:text-white"
                      :class="connBg(profile.color) + (connections.selectedId === profile.id ? '/60' : '/20')"
                      @click="activateProfile(profile)"
                    >
                      <Database class="size-4" />
                      <span class="sr-only">{{ profile.name }}</span>
                    </button>
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
                    <ContextMenuSeparator />
                    <template v-if="connections.isConnected(profile.id)">
                      <ContextMenuItem @click="disconnect(profile)">
                        Disconnect
                      </ContextMenuItem>
                    </template>
                    <template v-else>
                      <ContextMenuItem
                        :disabled="connections.isConnecting(profile.id)"
                        @click="connect(profile)"
                      >
                        <Spinner v-if="connections.isConnecting(profile.id)" />
                        Connect
                      </ContextMenuItem>
                    </template>
                  </ContextMenuContent>
                </ContextMenu>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </template>
    </SidebarContent>
    <SidebarFooter>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton
            tooltip="Settings"
            :is-active="route.path === '/settings'"
            class="px-2.5 md:px-2"
            @click="goSettings"
          >
            <Settings />
            <span>Settings</span>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarFooter>
  </Sidebar>
</template>
