<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Item,
  ItemContent,
  ItemFooter,
  ItemHeader,
  ItemSeparator,
  ItemTitle,
} from "@/components/ui/item";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { errorMessage } from "@/lib/errors";
import { AboutService } from "@/lib/wails";
import { useSettingsStore } from "@/stores/settings";
import { useUpdatesStore } from "@/stores/updates";

const settings = useSettingsStore();
const updates = useUpdatesStore();
const tokenDraft = ref("");
const tokenBusy = ref(false);

const canSaveToken = computed(() => tokenDraft.value.trim().length > 0 && !tokenBusy.value);
const canDeleteToken = computed(
  () => !tokenBusy.value && (updates.info.hasGitHubToken || tokenDraft.value.length > 0),
);
const checkDisabled = computed(
  () => updates.busy || updates.phase === "downloading",
);
const showInstallActions = computed(
  () => updates.phase === "available" || updates.phase === "ready",
);

onMounted(() => {
  void updates.loadInfo();
});

function onAutoInstall(value: unknown) {
  settings.setAutoInstallUpdates(Boolean(value));
}

async function saveToken() {
  const token = tokenDraft.value.trim();
  if (!token) {
    return;
  }
  tokenBusy.value = true;
  try {
    await AboutService.SetGitHubToken(token);
    tokenDraft.value = "";
    await updates.loadInfo();
    toast.success("GitHub token saved.");
  } catch (err) {
    toast.error(errorMessage(err));
  } finally {
    tokenBusy.value = false;
  }
}

async function deleteToken() {
  tokenBusy.value = true;
  try {
    await AboutService.ClearGitHubToken();
    tokenDraft.value = "";
    await updates.loadInfo();
    toast.success("GitHub token removed.");
  } catch (err) {
    toast.error(errorMessage(err));
  } finally {
    tokenBusy.value = false;
  }
}

function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return "";
  }
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
</script>

<template>
  <section class="mx-auto flex h-full w-full max-w-lg flex-col gap-8 overflow-auto py-2">
    <h1 class="text-lg font-medium">About</h1>

    <div class="flex items-center justify-between gap-4">
      <span class="text-sm">Version</span>
      <span class="text-muted-foreground font-mono text-sm">
        {{ updates.info.version || "…" }}
      </span>
    </div>

    <div class="flex items-center justify-between gap-4">
      <span class="text-sm">Architecture</span>
      <span class="text-muted-foreground font-mono text-sm">
        <template v-if="updates.info.goos">
          {{ updates.info.goos }}/{{ updates.info.goarch }}
        </template>
        <template v-else>…</template>
      </span>
    </div>

    <div class="flex items-center justify-between gap-4">
      <div class="min-w-0">
        <Label for="about-auto-install">Install updates automatically</Label>
        <p class="text-muted-foreground mt-1 text-xs">
          Checks hourly and downloads in the background. Restart is always manual.
        </p>
      </div>
      <Switch
        id="about-auto-install"
        :model-value="settings.autoInstallUpdates"
        @update:model-value="onAutoInstall"
      />
    </div>

    <Item variant="outline" class="min-w-0">
      <ItemHeader>
        <ItemTitle>Software update</ItemTitle>
      </ItemHeader>

      <ItemContent class="w-full min-w-0">
        <p class="text-muted-foreground text-sm">
          Last checked: {{ updates.lastCheckedLabel }}
        </p>
        <Button
          size="xs"
          variant="outline"
          class="mt-2 w-fit"
          :disabled="checkDisabled"
          @click="updates.check('manual')"
        >
          Check for updates
        </Button>

        <p
          v-if="updates.phase === 'idle' && settings.lastUpdateCheckAt"
          class="text-muted-foreground mt-2 text-xs"
        >
          LotusTab is up to date.
        </p>
        <p v-else-if="updates.phase === 'checking'" class="text-muted-foreground mt-2 text-xs">
          Checking for updates…
        </p>
        <div v-else-if="updates.available && updates.phase === 'available'" class="mt-2 flex flex-col gap-1">
          <p class="text-sm">
            Version {{ updates.available.version }} is available
            <span v-if="formatSize(updates.available.size)" class="text-muted-foreground">
              ({{ formatSize(updates.available.size) }})
            </span>
          </p>
          <p
            v-if="updates.available.notes"
            class="text-muted-foreground max-h-24 overflow-auto whitespace-pre-wrap text-xs"
          >
            {{ updates.available.notes }}
          </p>
        </div>
        <div v-else-if="updates.phase === 'downloading'" class="mt-2 flex flex-col gap-2">
          <p class="text-muted-foreground text-xs">
            Downloading{{ updates.available?.version ? ` ${updates.available.version}` : "" }}…
            <span v-if="updates.progressPercent">{{ updates.progressPercent }}%</span>
          </p>
          <div class="bg-muted h-1.5 overflow-hidden rounded-full">
            <div
              class="bg-primary h-full transition-[width]"
              :style="{ width: `${updates.progressPercent}%` }"
            />
          </div>
        </div>
        <p v-else-if="updates.phase === 'ready'" class="mt-2 text-sm">
          Update{{ updates.available?.version ? ` ${updates.available.version}` : "" }} is downloaded.
          Restart to apply it.
        </p>
        <p v-else-if="updates.phase === 'error'" class="text-destructive mt-2 text-xs">
          {{ updates.error || "Update failed" }}
        </p>
      </ItemContent>

      <ItemSeparator />

      <ItemContent class="w-full min-w-0">
        <Label for="about-github-token">GitHub token</Label>
        <Input
          id="about-github-token"
          v-model="tokenDraft"
          type="password"
          size="xs"
          class="mt-2 w-full min-w-0"
          autocomplete="off"
          :placeholder="updates.info.hasGitHubToken ? 'Saved in keychain' : 'PAT for private releases'"
        />
        <div class="mt-2 flex gap-2">
          <Button size="sm" :disabled="!canSaveToken" @click="saveToken">Save</Button>
          <Button size="sm" variant="outline" :disabled="!canDeleteToken" @click="deleteToken">
            Delete
          </Button>
        </div>
        <p class="text-muted-foreground mt-2 text-xs">Stored in the OS keychain.</p>
      </ItemContent>

      <ItemFooter v-if="showInstallActions" class="flex-wrap justify-start">
        <template v-if="updates.phase === 'available'">
          <Button size="sm" :disabled="updates.busy" @click="updates.installAndRestart()">
            Install and restart immediately
          </Button>
          <Button size="sm" variant="outline" :disabled="updates.busy" @click="updates.installLater()">
            Install and restart later
          </Button>
        </template>
        <Button
          v-else-if="updates.phase === 'ready'"
          size="sm"
          :disabled="updates.busy"
          @click="updates.restartNow()"
        >
          Restart now
        </Button>
      </ItemFooter>
    </Item>
  </section>
</template>
