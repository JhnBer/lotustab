export type CloseSide = "one" | "left" | "right" | "all";

export interface CloseTabRef {
  id: string;
  profileId: string;
}

export interface ClosePlan {
  closedIds: string[];
  activeId: string;
}

function tabsToClose(group: readonly CloseTabRef[], anchorAt: number, side: CloseSide): readonly CloseTabRef[] {
  switch (side) {
    case "one":
      return [group[anchorAt]!];
    case "left":
      return group.slice(0, anchorAt);
    case "right":
      return group.slice(anchorAt + 1);
    case "all":
      return group;
  }
}

function nextActive(group: readonly CloseTabRef[], activeId: string, closedIds: readonly string[]): string {
  const closed = new Set(closedIds);
  if (!closed.has(activeId)) {
    return activeId;
  }
  const activeAt = group.findIndex((tab) => tab.id === activeId);
  let left = "";
  let right = "";
  for (let index = 0; index < group.length; index++) {
    const tab = group[index]!;
    if (closed.has(tab.id) || index === activeAt) {
      continue;
    }
    if (index < activeAt) {
      left = tab.id;
    } else if (!right) {
      right = tab.id;
    }
  }
  return left || right;
}

export function planClose(
  tabs: readonly CloseTabRef[],
  activeId: string,
  anchorId: string,
  side: CloseSide,
): ClosePlan {
  const anchor = tabs.find((tab) => tab.id === anchorId);
  if (!anchor) {
    return { closedIds: [], activeId };
  }
  const group = tabs.filter((tab) => tab.profileId === anchor.profileId);
  const anchorAt = group.findIndex((tab) => tab.id === anchorId);
  const closedIds = tabsToClose(group, anchorAt, side).map((tab) => tab.id);
  if (closedIds.length === 0) {
    return { closedIds, activeId };
  }
  return { closedIds, activeId: nextActive(group, activeId, closedIds) };
}
