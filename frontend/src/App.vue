<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useColorMode } from "@vueuse/core";
import { useRoute, useRouter } from "vue-router";
import { toast } from "vue-sonner";
import { Events, Window } from "@wailsio/runtime";
import AppResizableLayout from "@/components/AppResizableLayout.vue";
import AppSidebarRail from "@/components/app-sidebar/AppSidebarRail.vue";
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
import { useBlockHistoryBackspace } from "@/composables/useBlockHistoryBackspace";
import { useConnectionTint } from "@/composables/useConnectionTint";
import { useExportData } from "@/composables/useExportData";
import { useWorkspaceShortcuts } from "@/composables/useWorkspaceShortcuts";
import ExportDataDialog from "@/features/export/ExportDataDialog.vue";
import { errorMessage } from "@/lib/errors";
import { WindowsService } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import { useQueryStore, type WorkspaceSnapshot } from "@/stores/query";
import { useUpdatesStore } from "@/stores/updates";

useColorMode({ initialValue: "dark" });

const route = useRoute();
const router = useRouter();
const connections = useConnectionsStore();
const queries = useQueryStore();
const updates = useUpdatesStore();
const { tintClass: headerTint } = useConnectionTint();
const { open: exportDataOpen, presetProfileId, presetDatabase, openExportData } =
  useExportData();
useWorkspaceShortcuts();
useBlockHistoryBackspace();

const pageTitle = computed(() => {
  const title = route.meta.title;
  return typeof title === "string" ? title : "LotusTab";
});

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
  await updates.loadInfo();
  try {
    await connections.refresh();
  } catch (err) {
    toast.error(errorMessage(err));
    updates.bindMainWindow();
  }
  Events.On("open:export-data", () => {
    openExportData();
  });
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
    updates.bindMainWindow();
  } catch {
    // Main window / non-Wails preview.
    updates.bindMainWindow();
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
    <ExportDataDialog
      v-model:open="exportDataOpen"
      :profile-id="presetProfileId"
      :database="presetDatabase"
    />
  </div>
</template>
