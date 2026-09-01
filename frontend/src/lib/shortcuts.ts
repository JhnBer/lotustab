export const WORKSPACE_SHORTCUTS = [
  { id: "tab.next", label: "Next tab", key: "Tab", shift: false },
  { id: "tab.prev", label: "Previous tab", key: "Tab", shift: true },
  { id: "tab.close", label: "Close tab", key: "W", shift: false },
  { id: "tab.reload", label: "Reload table", key: "R", shift: false },
  { id: "tab.new", label: "New query tab", key: "T", shift: false },
  { id: "window.new", label: "New window", key: "N", shift: false },
] as const;

export type ShortcutId = (typeof WORKSPACE_SHORTCUTS)[number]["id"];

export type ShortcutSpec = (typeof WORKSPACE_SHORTCUTS)[number];

function eventKey(event: KeyboardEvent): string {
  return event.key.length === 1 ? event.key.toUpperCase() : event.key;
}

export function matchWorkspaceShortcut(event: KeyboardEvent): ShortcutId | null {
  if (!(event.metaKey || event.ctrlKey)) {
    return null;
  }
  const key = eventKey(event);
  for (const spec of WORKSPACE_SHORTCUTS) {
    if (spec.shift !== event.shiftKey || spec.key !== key) {
      continue;
    }
    return spec.id;
  }
  return null;
}
