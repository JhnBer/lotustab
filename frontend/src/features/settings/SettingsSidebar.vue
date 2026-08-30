<script setup lang="ts">
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
} from "@/components/ui/sidebar";
</script>

<template>
  <Sidebar collapsible="none" class="bg-sidebar h-full min-h-0 w-full! overflow-hidden">
    <SidebarHeader>
      <div class="h-12 -m-2 flex items-center p-2">
        <div class="text-sm font-medium px-2 py-1">
          Settings
        </div>
      </div>
    </SidebarHeader>

    <SidebarSeparator class="m-0 -translate-y-full" />

    <SidebarContent class="overflow-hidden">
      <SidebarGroup>
        <SidebarGroupContent>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton :is-active="true">
                General
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>
  </Sidebar>
</template>
