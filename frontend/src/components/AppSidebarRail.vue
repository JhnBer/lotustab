<script setup lang="ts">
import { Database, Server } from "@lucide/vue";
import { storeToRefs } from "pinia";
import { useRoute, useRouter } from "vue-router";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
  useSidebar,
} from "@/components/ui/sidebar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { connBg } from "@/lib/colors";
import type { Profile } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";

const route = useRoute();
const router = useRouter();
const connections = useConnectionsStore();
const { setOpen } = useSidebar();
const { pinnedProfiles } = storeToRefs(connections);

function activateProfile(profile: Profile) {
  connections.select(profile.id);
  router.push("/query");
}
</script>

<template>
  <Sidebar
    collapsible="none"
    class="h-full w-[calc(var(--sidebar-width-icon)+1px)]! shrink-0 border-r"
  >
    <SidebarContent>
      <SidebarGroup>
        <SidebarGroupContent class="px-1.5 md:px-0">
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton
                tooltip="Connections"
                :is-active="route.path === '/connections'"
                class="px-2.5 md:px-2"
                @click="router.push('/connections'); setOpen(true)"
              >
                <Server />
                <span>Connections</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>

      <template v-if="pinnedProfiles.length > 0">
        <SidebarSeparator />
        <SidebarGroup>
          <SidebarGroupContent class="px-1.5 md:px-0">
            <SidebarMenu class="gap-1">
              <SidebarMenuItem v-for="profile in pinnedProfiles" :key="profile.id">
                <Tooltip>
                  <TooltipTrigger as-child>
                    <button
                      type="button"
                      class="mx-auto flex size-8 items-center justify-center rounded-md text-foreground ring-offset-background focus-visible:ring-2 focus-visible:outline-none dark:text-white"
                      :class="connBg(profile.color) + (connections.selectedId === profile.id ? '/60' : '/20')"
                      @click="activateProfile(profile)"
                    >
                      <Database class="size-4" />
                      <span class="sr-only">{{ profile.name }}</span>
                    </button>
                  </TooltipTrigger>
                  <TooltipContent side="right" align="center">
                    {{ profile.name }}
                  </TooltipContent>
                </Tooltip>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </template>
    </SidebarContent>
  </Sidebar>
</template>
