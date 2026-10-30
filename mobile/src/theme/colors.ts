// Warna mengikuti PRD bagian 8.2. Varian "Strong" dan "Text" dipakai untuk teks agar kontrasnya memadai.
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
  blobC: string;
  text: string;
  textMuted: string;
  surface: string;
  inputBg: string;
  inputBorder: string;
  glassBg: string;
  glassBgFallback: string;
  glassBorder: string;
  primary: string; // warna merek (dekoratif)
  primaryStrong: string; // latar tombol/chip dengan teks putih
  primaryText: string; // teks/tautan berwarna biru
  secondary: string;
  secondaryStrong: string;
  danger: string;
  warning: string;
  income: string;
  expense: string;
}

export const lightColors: Colors = {
  bgGradient: ["#EAF2FF", "#DDE8FF", "#EDE9FE"],
  blobA: "rgba(79,124,255,0.30)",
  blobB: "rgba(139,92,246,0.25)",
  blobC: "rgba(34,211,238,0.30)",
  text: "#0F172A",
  textMuted: "#475569",
  surface: "#FFFFFF",
  inputBg: "#F1F5FF",
  inputBorder: "#CBD5E1",
  glassBg: "rgba(255,255,255,0.45)",
  glassBgFallback: "rgba(255,255,255,0.72)",
  glassBorder: "rgba(255,255,255,0.8)",
  primary: palette.primary,
  primaryStrong: "#3B63E8",
  primaryText: "#2F5BE0",
  secondary: palette.secondary,
  secondaryStrong: "#7C3AED",
  danger: "#BE123C",
  warning: "#92400E",
  income: "#047857",
  expense: "#BE123C",
};

export const darkColors: Colors = {
  bgGradient: ["#0B1220", "#141B3A", "#1E1B4B"],
  blobA: "rgba(79,124,255,0.35)",
  blobB: "rgba(139,92,246,0.30)",
  blobC: "rgba(34,211,238,0.20)",
  text: "#F1F5F9",
  textMuted: "#94A3B8",
  surface: "#111A2E",
  inputBg: "#0B1220",
  inputBorder: "#2B3A5C",
  glassBg: "rgba(255,255,255,0.07)",
  glassBgFallback: "rgba(255,255,255,0.12)",
  glassBorder: "rgba(255,255,255,0.16)",
  primary: palette.primary,
  primaryStrong: "#3B63E8",
  primaryText: "#7C9BFF",
  secondary: palette.secondary,
  secondaryStrong: "#7C3AED",
  danger: palette.expense,
  warning: "#FBBF24",
  income: palette.income,
  expense: palette.expense,
};
