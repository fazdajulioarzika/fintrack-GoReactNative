import { useThemeContext } from "@/theme/ThemeContext";

export function useTheme() {
  const { isDark, colors } = useThemeContext();
  return { isDark, colors };
}

export function useThemePreference() {
  const { pref, setPref } = useThemeContext();
  return { pref, setPref };
}
