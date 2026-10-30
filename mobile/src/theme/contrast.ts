function toLinear(v: number): number {
  const c = v / 255;
  return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
}

function luminance(hex: string): number | null {
  const m = /^#([0-9a-f]{6})$/i.exec(hex);
  if (!m) return null;
  const n = parseInt(m[1], 16);
  return (
    0.2126 * toLinear((n >> 16) & 255) +
    0.7152 * toLinear((n >> 8) & 255) +
    0.0722 * toLinear(n & 255)
  );
}

const DARK = "#0B1220";
const DARK_LUM = luminance(DARK) ?? 0;

// Pilih warna teks (putih atau gelap) yang kontrasnya paling tinggi di atas latar `bg`.
export function onColor(bg: string): string {
  const l = luminance(bg);
  if (l === null) return "#FFFFFF";
  const white = 1.05 / (l + 0.05);
  const dark = (l + 0.05) / (DARK_LUM + 0.05);
  return white >= dark ? "#FFFFFF" : DARK;
}
