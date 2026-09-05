import { toast } from "vue-sonner";
import { useRouter } from "vue-router";
import { errorMessage, isCanceled } from "@/lib/errors";
import type { Profile } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import { useQueryStore } from "@/stores/query";

export type ConnectOptions = {
  /** Cancel an in-flight connect instead of starting another. Default true. */
  cancelIfConnecting?: boolean;
  /** Select the profile, open a query tab, and navigate to /query. */
  openQuery?: boolean;
  /** Navigate to /query after a successful connect. */
  navigate?: boolean;
};

export type ActivateOptions = {
  /** Open (or focus) a query tab for the profile. */
  openTab?: boolean;
};

export function useProfileActions() {
  const connections = useConnectionsStore();
  const queries = useQueryStore();
  const router = useRouter();

  function activateProfile(profile: Profile, options?: ActivateOptions) {
    connections.select(profile.id);
    if (options?.openTab) {
      queries.openForProfile(profile.id, profile.name);
    }
    void router.push("/query");
  }

  async function togglePin(profile: Profile) {
    try {
      await connections.togglePinned(profile.id);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  }

  async function unpin(profile: Profile) {
    try {
      await connections.setPinned(profile.id, false);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  }

  async function connect(profile: Profile, options?: ConnectOptions) {
    if (options?.cancelIfConnecting !== false && connections.isConnecting(profile.id)) {
      connections.cancelConnect(profile.id);
      return;
    }
    try {
      await connections.connect(profile.id);
      if (!connections.isConnected(profile.id)) {
        return;
      }
      toast.success(`Connected to ${profile.name}`);
      if (options?.openQuery) {
        connections.select(profile.id);
        queries.openForProfile(profile.id, profile.name);
        await router.push("/query");
      } else if (options?.navigate) {
        await router.push("/query");
      }
    } catch (err) {
      if (isCanceled(err)) {
        return;
      }
      toast.error(errorMessage(err));
    }
  }

  async function disconnect(profile: Profile) {
    try {
      await connections.disconnect(profile.id);
      toast.info(`Disconnected from ${profile.name}`);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  }

  async function clone(profile: Profile) {
    try {
      await connections.clone(profile.id);
      toast.success(`Cloned ${profile.name}`);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  }

  async function remove(profile: Profile) {
    try {
      await connections.remove(profile.id);
      toast.success(`Deleted ${profile.name}`);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  }

  async function openInNewWindow(profile: Profile) {
    try {
      await connections.openInNewWindow(profile);
      if (!connections.selectedId) {
        await router.push("/connections");
      }
    } catch (err) {
      toast.error(errorMessage(err));
    }
  }

  return {
    activateProfile,
    togglePin,
    unpin,
    connect,
    disconnect,
    clone,
    remove,
    openInNewWindow,
  };
}
