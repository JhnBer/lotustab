export const CONNECTION_COLORS = [
  "red",
  "orange",
  "amber",
  "green",
  "teal",
  "sky",
  "blue",
  "indigo",
  "violet",
  "pink",
  "slate",
] as const;

export type ConnectionColor = (typeof CONNECTION_COLORS)[number];

export const DEFAULT_CONNECTION_COLOR: ConnectionColor = "blue";

export function connectionColor(color?: string | null): ConnectionColor {
  if (color && (CONNECTION_COLORS as readonly string[]).includes(color)) {
    return color as ConnectionColor;
  }
  return DEFAULT_CONNECTION_COLOR;
}

export function colorBgClass(color?: string | null): string {
  return 'bg-' + connectionColor(color) + '-500';
}

export function colorCssVar(color?: string | null): string {
  return `var(--color-${connectionColor(color)}-500)`;
}

export function colorTintClass(active = false): string {
  return active ? "bg-(--conn-color)/60" : "bg-(--conn-color)/20";
}
