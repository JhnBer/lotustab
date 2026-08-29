// Tailwind hues. If you add one, also add it in style.css `@source inline`.
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

export function connColor(color?: string | null): ConnectionColor {
  if (color && (CONNECTION_COLORS as readonly string[]).includes(color)) {
    return color as ConnectionColor;
  }
  return "blue";
}

export function connBg(color?: string | null): string {
  return `bg-${connColor(color)}-500`;
}

export function connTint(color?: string | null): string {
  const bg = connBg(color);
  return `${bg}/20 dark:${bg}/60`;
}
