<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { Browser } from "@wailsio/runtime";
import { Button } from "@/components/ui/button";
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
import { useSettingsStore } from "@/stores/settings";
import { useUpdatesStore } from "@/stores/updates";

/** Matches build/config.yml productName; no backend field. */
const productName = "LotusTab";
/** Public asset; keep as a runtime URL so Vite does not rewrite it. */
const logoSrc = "/logo.svg";
const githubURL = "https://github.com/JhnBer/lotustab";
const licenseURL = "https://github.com/JhnBer/lotustab/blob/main/LICENSE";

const settings = useSettingsStore();
const { t } = useI18n();
const updates = useUpdatesStore();

const versionLine = computed(() => {
  const { version, goos, goarch } = updates.info;
  if (!version || !goos || !goarch) {
    return "…";
  }
  return `${version} · ${goos}/${goarch}`;
});

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

function openExternal(url: string) {
  void Browser.OpenURL(url);
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
  <div class="h-full overflow-y-auto">
  <section class="mx-auto flex w-full max-w-lg flex-col gap-8 px-4 py-4">
    <h1 class="text-lg font-medium">{{ t('about.title') }}</h1>

    <div class="flex flex-col items-center gap-1.5">
      <img :src="logoSrc" alt="" width="64" height="64" class="size-16" />
      <p class="text-base font-medium">{{ productName }}</p>
      <p class="text-muted-foreground font-mono text-sm">{{ versionLine }}</p>
      <p class="text-muted-foreground mt-1 text-xs">
        <a
          :href="githubURL"
          class="hover:text-foreground underline-offset-4 hover:underline"
          @click.prevent="openExternal(githubURL)"
        >{{ t('about.github') }}</a>
        <span class="px-1.5">·</span>
        <a
          :href="licenseURL"
          class="hover:text-foreground underline-offset-4 hover:underline"
          @click.prevent="openExternal(licenseURL)"
        >{{ t('about.license') }}</a>
      </p>
    </div>

    <Item variant="outline" class="min-w-0">
      <ItemHeader>
        <ItemTitle>{{ t('about.softwareUpdate') }}</ItemTitle>
      </ItemHeader>

      <ItemContent class="w-full min-w-0">
        <p class="text-muted-foreground text-sm">
          {{ t('about.lastChecked', { when: updates.lastCheckedLabel }) }}
        </p>
        <Button
          size="xs"
          variant="outline"
          class="mt-2 w-fit"
          :disabled="checkDisabled"
          @click="updates.check('manual')"
        >
          {{ t('about.check') }}
        </Button>

        <p
          v-if="updates.phase === 'idle' && settings.lastUpdateCheckAt"
          class="text-muted-foreground mt-2 text-xs"
        >
          {{ t('about.upToDate') }}
        </p>
        <p v-else-if="updates.phase === 'checking'" class="text-muted-foreground mt-2 text-xs">
          {{ t('about.checking') }}
        </p>
        <div v-else-if="updates.available && updates.phase === 'available'" class="mt-2 flex flex-col gap-1">
          <p class="text-sm">
            {{ t('about.versionAvailable', { version: updates.available.version }) }}
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
            {{ t('about.downloading', { version: updates.available?.version ? ` ${updates.available.version}` : '' }) }}
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
          {{ t('about.ready', { version: updates.available?.version ? ` ${updates.available.version}` : '' }) }}
        </p>
        <p v-else-if="updates.phase === 'error'" class="text-destructive mt-2 text-xs">
          {{ updates.error || t('about.failed') }}
        </p>
      </ItemContent>

      <ItemSeparator />

      <ItemContent class="w-full min-w-0">
        <div class="flex items-center justify-between gap-4">
          <div class="min-w-0">
            <Label for="about-auto-install">{{ t('about.autoInstall') }}</Label>
            <p class="text-muted-foreground mt-1 text-xs">
              {{ t('about.autoInstallHelp') }}
            </p>
          </div>
          <Switch
            id="about-auto-install"
            :model-value="settings.autoInstallUpdates"
            @update:model-value="onAutoInstall"
          />
        </div>
      </ItemContent>

      <ItemFooter v-if="showInstallActions || updates.info.updateError" class="flex-wrap justify-start">
        <template v-if="updates.phase === 'available'">
          <Button size="sm" :disabled="updates.busy" @click="updates.installAndRestart()">
            {{ t('about.installNow') }}
          </Button>
          <Button size="sm" variant="outline" :disabled="updates.busy" @click="updates.installLater()">
            {{ t('about.installLater') }}
          </Button>
        </template>
        <Button
          v-else-if="updates.phase === 'ready' || updates.info.updateError"
          size="sm"
          :disabled="updates.busy"
          @click="updates.restartNow()"
        >
          {{ t('about.restartNow') }}
        </Button>
      </ItemFooter>
    </Item>
  </section>
  </div>
</template>
