import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useColorScheme } from "react-native";

import { prefStore } from "@/lib/storage";
import { darkColors, lightColors, type Colors } from "@/theme/colors";

export type ThemePref = "system" | "light" | "dark";
const KEY = "theme";

interface ThemeValue {
  pref: ThemePref;
  setPref: (p: ThemePref) => void;
  isDark: boolean;
  colors: Colors;
}

const ThemeContext = createContext<ThemeValue | null>(null);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const system = useColorScheme();
  const [pref, setPrefState] = useState<ThemePref>("system");
  const [ready, setReady] = useState(false);

  useEffect(() => {
    prefStore
      .get(KEY)
      .then((v) => {
        if (v === "light" || v === "dark" || v === "system") setPrefState(v);
      })
      .catch(() => {})
      .finally(() => setReady(true));
  }, []);

  const setPref = useCallback((p: ThemePref) => {
    setPrefState(p);
    prefStore.set(KEY, p).catch(() => {});
  }, []);

  const isDark = pref === "system" ? system === "dark" : pref === "dark";
  const value = useMemo(
    () => ({
      pref,
      setPref,
      isDark,
      colors: isDark ? darkColors : lightColors,
    }),
    [pref, setPref, isDark]
  );

  // Tunggu preferensi terbaca agar tidak ada kedipan tema yang salah saat aplikasi dibuka
  if (!ready) return null;
  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  );
}

export function useThemeContext(): ThemeValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useTheme harus dipakai di dalam ThemeProvider");
  return ctx;
}
