import type { Ref } from "vue";
import type { QueryTab, TableTab, WorkspaceTab } from "@/features/editor/tabs";

export type QueryStoreCtx = {
  tabs: Ref<WorkspaceTab[]>;
  inflight: Map<string, { cancel: () => void }>;
  cancel: (id: string) => void;
  storedPageSize: () => number;
  tableTab: (id: string) => TableTab | undefined;
  queryTab: (id: string) => QueryTab | undefined;
};
