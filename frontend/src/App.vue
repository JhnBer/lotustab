<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { toast } from "vue-sonner";
import { Window } from "@wailsio/runtime";
import AppResizableLayout from "@/components/AppResizableLayout.vue";
import AppSidebarRail from "@/components/AppSidebarRail.vue";
import ModeToggle from "@/components/ModeToggle.vue";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Separator } from "@/components/ui/separator";
import {
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { Toaster } from "@/components/ui/sonner";
import { errorMessage } from "@/lib/errors";
import { useConnectionsStore } from "@/stores/connections";
import { useQueryStore } from "@/stores/query";

const route = useRoute();
const router = useRouter();
const connections = useConnectionsStore();
const queries = useQueryStore();

const pageTitle = computed(() => {
  const title = route.meta.title;
  return typeof title === "string" ? title : "LotusTab";
});

onMounted(async () => {
  try {
    await connections.refresh();
  } catch (err) {
    toast.error(errorMessage(err));
    return;
  }
  try {
    const name = await Window.Name();
    if (!name.startsWith("conn-")) {
      return;
    }
    const id = name.slice("conn-".length);
    const profile = connections.profiles.find((p) => p.id === id);
    connections.select(id);
    if (profile) {
      queries.openForProfile(id, profile.name);
    }
    await router.replace("/query");
  } catch {
    // Main window / non-Wails preview.
  }
});
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <div
      class="drag-region bg-background border-b shrink-0"
      :style="{ height: 'var(--titlebar-height)' }"
    />
    <SidebarProvider
      class="flex h-full min-h-0 flex-1 overflow-hidden"
    >
      <AppSidebarRail />
      <AppResizableLayout>
        <header class="drag-region bg-background sticky top-0 flex h-12 shrink-0 items-center gap-2 border-b px-4 transition-[width,height] ease-linear">
          <div class="flex min-w-0 flex-1 items-center gap-2" style="--wails-draggable: no-drag">
            <SidebarTrigger class="-ml-1" />
            <Separator
              orientation="vertical"
              class="mr-2 data-[orientation=vertical]:h-12"
            />
            <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem class="hidden md:block">
                  <BreadcrumbLink as-child>
                    <RouterLink to="/connections">LotusTab</RouterLink>
                  </BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator class="hidden md:block" />
                <BreadcrumbItem>
                  <BreadcrumbPage>{{ pageTitle }}</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
          <div style="--wails-draggable: no-drag">
            <ModeToggle />
          </div>
        </header>
        <div class="flex min-h-0 flex-1 flex-col overflow-hidden p-4">
          <RouterView v-slot="{ Component }">
            <component :is="Component" class="h-full min-h-0 flex-1" />
          </RouterView>
        </div>
      </AppResizableLayout>
      <Toaster rich-colors position="bottom-right" />
    </SidebarProvider>
  </div>
</template>
