<script setup lang="ts">
import { storeToRefs } from "pinia";
import {
  Sidebar,
  SidebarContent,
  SidebarSeparator,
} from "@/components/ui/sidebar";
import AppSidebarHeader from "./AppSidebarHeader.vue";
import AppSidebarPane from "./AppSidebarPane.vue";
import AppSidebarRecent from "./AppSidebarRecent.vue";
import { useConnectionsStore } from "@/stores/connections";

const connections = useConnectionsStore();
const { selected } = storeToRefs(connections);
</script>

<template>
  <Sidebar collapsible="none" class="bg-sidebar h-full min-h-0 w-full! overflow-hidden">
    <AppSidebarHeader />

    <SidebarSeparator class="m-0 -translate-y-full" />

    <SidebarContent class="overflow-hidden">
      <AppSidebarRecent v-if="!selected" />
      <AppSidebarPane v-else />
    </SidebarContent>
  </Sidebar>
</template>
