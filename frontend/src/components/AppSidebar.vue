<script setup lang="ts">
import type { SidebarProps } from "@/components/ui/sidebar";
import { Database } from "@lucide/vue";
import { useRoute, useRouter } from "vue-router";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
  useSidebar,
} from "@/components/ui/sidebar";
import type { Profile } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import { useQueryStore } from "@/stores/query";

const props = withDefaults(defineProps<SidebarProps>(), {
  collapsible: "icon",
});

const route = useRoute();
const router = useRouter();
const connections = useConnectionsStore();
const queries = useQueryStore();
const { setOpen } = useSidebar();

function openProfile(profile: Profile) {
  connections.select(profile.id);
  queries.openForProfile(profile.id, profile.name);
  router.push("/query");
}
</script>

<template>
  <Sidebar
    class="overflow-hidden *:data-[sidebar=sidebar]:flex-row"
    v-bind="props"
  >
    <!-- Icon rail -->
    <Sidebar
      collapsible="none"
      class="w-[calc(var(--sidebar-width-icon)+1px)]! border-r"
    >
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent class="px-1.5 md:px-0">
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  tooltip="Connections"
                  :is-active="true"
                  class="px-2.5 md:px-2"
                  @click="setOpen(true)"
                >
                  <Database />
                  <span>Connections</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>

    <!-- Connection list -->
    <Sidebar collapsible="none" class="hidden flex-1 md:flex">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              as-child
              tooltip="Connections"
              :is-active="route.path === '/connections'"
            >
              <RouterLink to="/connections">
                <Database />
                <span>Connections</span>
              </RouterLink>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarSeparator />

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem v-if="connections.profiles.length === 0">
                <SidebarMenuButton class="text-muted-foreground pointer-events-none">
                  <span>No connections yet</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem
                v-for="profile in connections.profiles"
                :key="profile.id"
              >
                <SidebarMenuButton
                  :tooltip="profile.name"
                  :is-active="connections.selectedId === profile.id && route.path === '/query'"
                  @click="openProfile(profile)"
                >
                  <span
                    class="size-1.5 shrink-0 rounded-full"
                    :class="connections.isConnected(profile.id) ? 'bg-emerald-500' : 'bg-muted-foreground/40'"
                  />
                  <span>{{ profile.name }}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  </Sidebar>
</template>
