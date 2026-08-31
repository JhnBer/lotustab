<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useColorMode } from "@vueuse/core";
import { useRoute, useRouter } from "vue-router";
import { toast } from "vue-sonner";
import { Events, Window } from "@wailsio/runtime";
import AppResizableLayout from "@/components/AppResizableLayout.vue";
import AppSidebarRail from "@/components/AppSidebarRail.vue";
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
import { connTint } from "@/lib/colors";
import { WindowsService } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import { useQueryStore, type WorkspaceSnapshot } from "@/stores/query";
import { useSettingsStore } from "@/stores/settings";

useColorMode({ initialValue: "dark" });

const route = useRoute();
const router = useRouter();
const connections = useConnectionsStore();
const queries = useQueryStore();
const settings = useSettingsStore();

const pageTitle = computed(() => {
  const title = route.meta.title;
  return typeof title === "string" ? title : "LotusTab";
});

const headerTint = computed(() =>
  settings.connectionTint && connections.selected && !route.path.startsWith("/settings")
    ? connTint(connections.selected.color)
    : "",
);

function restoreHandoff(raw: string | null | undefined): boolean {
  if (!raw) {
    return false;
  }
  try {
    queries.restore(JSON.parse(raw) as WorkspaceSnapshot);
    return true;
  } catch {
    return false;
  }
}

onMounted(async () => {
  try {
    await connections.refresh();
  } catch (err) {
    toast.error(errorMessage(err));
    return;
  }
  try {
    const name = await Window.Name();
    if (name.startsWith("conn-")) {
      const id = name.slice("conn-".length);
      const profile = connections.profiles.find((p) => p.id === id);
      connections.select(id);
      if (!restoreHandoff(await WindowsService.TakeHandoff(id)) && profile) {
        queries.openForProfile(id, profile.name);
      }
      await router.replace("/query");
      Events.On("workspace:handoff", () => {
        void WindowsService.TakeHandoff(id).then((next) => {
          restoreHandoff(next);
        });
      });
      return;
    }
  } catch {
    // Main window / non-Wails preview.
  }
  if (connections.selectedId) {
    await router.replace("/query");
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
        <header
          class="drag-region sticky top-0 flex h-12 shrink-0 items-center gap-2 border-b px-4 transition-[width,height] ease-linear"
          :class="headerTint || 'bg-background'"
        >
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
