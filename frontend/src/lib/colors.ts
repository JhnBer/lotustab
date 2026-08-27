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

export const colorBg: Record<ConnectionColor, string> = {
  red: "bg-red-500",
  orange: "bg-orange-500",
  amber: "bg-amber-500",
  green: "bg-green-500",
  teal: "bg-teal-500",
  sky: "bg-sky-500",
  blue: "bg-blue-500",
  indigo: "bg-indigo-500",
  violet: "bg-violet-500",
  pink: "bg-pink-500",
  slate: "bg-slate-500",
};

export function connectionColor(color?: string | null): ConnectionColor {
  if (color && (CONNECTION_COLORS as readonly string[]).includes(color)) {
    return color as ConnectionColor;
  }
  return DEFAULT_CONNECTION_COLOR;
}

export function colorBgClass(color?: string | null): string {
  return colorBg[connectionColor(color)];
}
