import { defineStore } from "pinia";
import { computed, ref, watch } from "vue";
import { Events } from "@wailsio/runtime";
import { toast } from "vue-sonner";
import { errorMessage } from "@/lib/errors";
import { AboutService } from "@/lib/wails";
import type { AboutInfo, UpdateCheckResult } from "@/lib/wails";
import { useSettingsStore, type UpdateCheckSource } from "@/stores/settings";

export type UpdatePhase = "idle" | "checking" | "available" | "downloading" | "ready" | "error";

const CHECK_INTERVAL_MS = 60 * 60 * 1000;

const EMPTY_INFO: AboutInfo = {
  version: "",
  goos: "",
  goarch: "",
  updatesEnabled: false,
  state: "",
};

type ProgressPayload = {
  written?: number;
  total?: number;
};

type ReleasePayload = {
  version?: string;
  name?: string;
  notes?: string;
  artifact?: { filename?: string; size?: number };
};

type ErrorPayload = {
  message?: string;
};

function unwrap<T>(raw: unknown): T | null {
  const data = Array.isArray(raw) ? raw[0] : raw;
  if (!data || typeof data !== "object") {
    return null;
  }
  return data as T;
}

function releaseFromEvent(raw: unknown): UpdateCheckResult | null {
  const rel = unwrap<ReleasePayload>(raw);
  if (!rel || typeof rel.version !== "string" || !rel.version) {
    return null;
  }
  return {
    available: true,
    version: rel.version,
    name: typeof rel.name === "string" ? rel.name : "",
    notes: typeof rel.notes === "string" ? rel.notes : "",
    filename: typeof rel.artifact?.filename === "string" ? rel.artifact.filename : "",
    size: Number(rel.artifact?.size) || 0,
  };
}

export const useUpdatesStore = defineStore("updates", () => {
  const settings = useSettingsStore();

  const info = ref<AboutInfo>({ ...EMPTY_INFO });
  const available = ref<UpdateCheckResult | null>(null);
  const phase = ref<UpdatePhase>("idle");
  const error = ref("");
  const written = ref(0);
  const total = ref(0);
  const busy = ref(false);

  let hourly: ReturnType<typeof setInterval> | null = null;
  let eventsBound = false;
  let mainBound = false;

  const progressPercent = computed(() => {
    if (total.value <= 0) {
      return 0;
    }
    return Math.min(100, Math.round((written.value / total.value) * 100));
  });

  const lastCheckedLabel = computed(() => {
    if (!settings.lastUpdateCheckAt) {
      return "Never";
    }
    const when = new Date(settings.lastUpdateCheckAt).toLocaleString();
    if (!settings.lastUpdateCheckSource) {
      return when;
    }
    return `${when} (${settings.lastUpdateCheckSource})`;
  });

  function bindEvents() {
    if (eventsBound) {
      return;
    }
    eventsBound = true;

    Events.On("wails:updater:check-started", () => {
      if (phase.value !== "ready" && phase.value !== "downloading") {
        phase.value = "checking";
      }
    });

    Events.On("wails:updater:update-available", (ev) => {
      const next = releaseFromEvent(ev.data);
      if (next) {
        available.value = next;
      }
      if (phase.value !== "ready" && phase.value !== "downloading") {
        phase.value = "available";
      }
    });

    Events.On("wails:updater:no-update", () => {
      if (phase.value !== "ready" && phase.value !== "downloading") {
        phase.value = "idle";
        available.value = null;
      }
    });

    Events.On("wails:updater:download-started", (ev) => {
      phase.value = "downloading";
      written.value = 0;
      total.value = 0;
      const next = releaseFromEvent(ev.data);
      if (next) {
        available.value = next;
      }
    });

    Events.On("wails:updater:download-progress", (ev) => {
      const next = unwrap<ProgressPayload>(ev.data);
      if (!next) {
        return;
      }
      written.value = Number(next.written) || 0;
      total.value = Number(next.total) || 0;
      phase.value = "downloading";
    });

    Events.On("wails:updater:update-ready", (ev) => {
      const next = releaseFromEvent(ev.data);
      if (next) {
        available.value = next;
      }
      phase.value = "ready";
      busy.value = false;
    });

    Events.On("wails:updater:error", (ev) => {
      const next = unwrap<ErrorPayload>(ev.data);
      error.value = typeof next?.message === "string" && next.message ? next.message : "Update failed";
      phase.value = "error";
      busy.value = false;
    });
  }

  async function loadInfo() {
    try {
      info.value = await AboutService.Info();
      if (info.value.state === "ready" && phase.value !== "ready") {
        phase.value = "ready";
      }
    } catch (err) {
      error.value = errorMessage(err);
    }
  }

  async function check(source: UpdateCheckSource) {
    if (!info.value.updatesEnabled || busy.value) {
      return;
    }
    if (phase.value === "downloading" || phase.value === "ready") {
      return;
    }
    busy.value = true;
    error.value = "";
    phase.value = "checking";
    try {
      const result = await AboutService.Check();
      settings.recordUpdateCheck(source);
      if (result.available) {
        available.value = result;
        phase.value = "available";
        if (source === "automatic" && settings.autoInstallUpdates) {
          await installLater();
          if (!error.value) {
            toast.success(`Update ${result.version} downloaded. Restart when ready.`);
          }
        }
      } else {
        available.value = null;
        phase.value = "idle";
      }
    } catch (err) {
      error.value = errorMessage(err);
      phase.value = "error";
    } finally {
      busy.value = false;
    }
  }

  async function installLater() {
    if (!info.value.updatesEnabled || phase.value === "ready") {
      return;
    }
    busy.value = true;
    error.value = "";
    phase.value = "downloading";
    try {
      await AboutService.Install();
      phase.value = "ready";
    } catch (err) {
      error.value = errorMessage(err);
      phase.value = "error";
    } finally {
      busy.value = false;
    }
  }

  async function installAndRestart() {
    if (!info.value.updatesEnabled) {
      return;
    }
    busy.value = true;
    error.value = "";
    if (phase.value !== "ready") {
      phase.value = "downloading";
    }
    try {
      await AboutService.InstallAndRestart();
    } catch (err) {
      error.value = errorMessage(err);
      phase.value = "error";
      busy.value = false;
    }
  }

  async function restartNow() {
    if (!info.value.updatesEnabled) {
      return;
    }
    busy.value = true;
    try {
      await AboutService.Restart();
    } catch (err) {
      error.value = errorMessage(err);
      phase.value = "error";
      busy.value = false;
    }
  }

  function stopHourly() {
    if (hourly) {
      clearInterval(hourly);
      hourly = null;
    }
  }

  function startHourly() {
    stopHourly();
    hourly = setInterval(() => {
      void check("automatic");
    }, CHECK_INTERVAL_MS);
  }

  function bindMainWindow() {
    if (mainBound) {
      return;
    }
    mainBound = true;
    bindEvents();
    watch(
      () => [settings.autoInstallUpdates, info.value.updatesEnabled] as const,
      ([autoOn, enabled]) => {
        if (autoOn && enabled) {
          startHourly();
          void check("automatic");
        } else {
          stopHourly();
        }
      },
      { immediate: true },
    );
  }

  return {
    info,
    available,
    phase,
    error,
    written,
    total,
    busy,
    progressPercent,
    lastCheckedLabel,
    loadInfo,
    check,
    installLater,
    installAndRestart,
    restartNow,
    bindMainWindow,
  };
});
