// Warna mengikuti PRD bagian 8.2
export const palette = {
  primary: "#4F7CFF",
  secondary: "#8B5CF6",
  accent: "#22D3EE",
  income: "#34D399",
  expense: "#FB7185",
};

export interface Colors {
  bgGradient: readonly [string, string, ...string[]];
  blobA: string;
  blobB: string;
  text: string;
  textMuted: string;
  surface: string;
  inputBg: string;
  inputBorder: string;
  glassBg: string;
  glassBgFallback: string;
  glassBorder: string;
  primary: string;
  secondary: string;
  danger: string;
  income: string;
  expense: string;
}

export const lightColors: Colors = {
  bgGradient: ["#EAF2FF", "#DDE8FF", "#EDE9FE"],
  blobA: "rgba(79,124,255,0.30)",
  blobB: "rgba(139,92,246,0.25)",
  text: "#0F172A",
  textMuted: "#475569",
  surface: "#FFFFFF",
  inputBg: "#F1F5FF",
  inputBorder: "#CBD5E1",
  glassBg: "rgba(255,255,255,0.45)",
  glassBgFallback: "rgba(255,255,255,0.72)",
  glassBorder: "rgba(255,255,255,0.8)",
  primary: palette.primary,
  secondary: palette.secondary,
  danger: "#E11D48",
  income: "#059669",
  expense: "#E11D48",
};

export const darkColors: Colors = {
  bgGradient: ["#0B1220", "#141B3A", "#1E1B4B"],
  blobA: "rgba(79,124,255,0.35)",
  blobB: "rgba(139,92,246,0.30)",
  text: "#F1F5F9",
  textMuted: "#94A3B8",
  surface: "#111A2E",
  inputBg: "#0B1220",
  inputBorder: "#2B3A5C",
  glassBg: "rgba(255,255,255,0.07)",
  glassBgFallback: "rgba(255,255,255,0.12)",
  glassBorder: "rgba(255,255,255,0.16)",
  primary: palette.primary,
  secondary: palette.secondary,
  danger: palette.expense,
  income: palette.income,
  expense: palette.expense,
};
