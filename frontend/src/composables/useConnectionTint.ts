import { computed } from "vue";
import { useRoute } from "vue-router";
import { connTint } from "@/lib/colors";
import { useConnectionsStore } from "@/stores/connections";
import { useSettingsStore } from "@/stores/settings";

export function useConnectionTint() {
  const route = useRoute();
  const connections = useConnectionsStore();
  const settings = useSettingsStore();

  const tintClass = computed(() =>
    settings.connectionTint && connections.selected && !route.path.startsWith("/settings")
      ? connTint(connections.selected.color)
      : "",
  );

  return { tintClass };
}
