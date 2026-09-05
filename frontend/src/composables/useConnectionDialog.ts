import { ref } from "vue";
import type { Profile } from "@/lib/wails";

export function useConnectionDialog() {
  const dialogOpen = ref(false);
  const editing = ref<Profile | null>(null);

  function openCreate() {
    editing.value = null;
    dialogOpen.value = true;
  }

  function openEdit(profile: Profile) {
    editing.value = profile;
    dialogOpen.value = true;
  }

  return { dialogOpen, editing, openCreate, openEdit };
}
