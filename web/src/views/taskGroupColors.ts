import type { CSSProperties } from "react";

export function taskGroupColors(color: string): CSSProperties | undefined {
  if (!color) return undefined;
  return {
    backgroundColor: `color-mix(in srgb, var(--surface), ${color} 14%)`,
    borderColor: `color-mix(in srgb, var(--border-strong), ${color} 65%)`,
  };
}
