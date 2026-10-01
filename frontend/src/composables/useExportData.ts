import { ref } from "vue";

const open = ref(false);
const presetProfileId = ref("");
const presetDatabase = ref("");

export function useExportData() {
  function openExportData(profileId?: string, database?: string) {
    presetProfileId.value = profileId ?? "";
    presetDatabase.value = database ?? "";
    open.value = true;
  }

  return { open, presetProfileId, presetDatabase, openExportData };
}
