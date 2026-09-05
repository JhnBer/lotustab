export type {
  DraftColumn,
  QueryTab,
  SortDir,
  TableTab,
  TableView,
  WorkspaceSnapshot,
  WorkspaceTab,
} from "@/features/editor/tabs";
export { isQueryTab, isTableTab } from "@/features/editor/tabs";
export { useWorkspaceStore as useQueryStore } from "@/stores/workspace";
