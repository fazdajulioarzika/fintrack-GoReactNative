import { BlurView } from "expo-blur";
import React from "react";
import { Platform, StyleProp, View, ViewStyle } from "react-native";

import { useTheme } from "@/theme/useTheme";

interface Props {
  children: React.ReactNode;
  style?: StyleProp<ViewStyle>;
  solid?: boolean; // latar solid untuk formulir (PRD 8.3)
}

export function GlassCard({ children, style, solid = false }: Props) {
  const { colors, isDark } = useTheme();
  const base: ViewStyle = {
    borderRadius: 24,
    borderWidth: 1,
    borderColor: colors.glassBorder,
    overflow: "hidden",
  };

  if (solid) {
    return (
      <View style={[base, { backgroundColor: colors.surface }, style]}>
        {children}
      </View>
    );
  }
  // Blur asli hanya di iOS. Android (blur masih eksperimental) dan web memakai fallback semi-transparan.
  if (Platform.OS === "ios") {
    return (
      <BlurView
        intensity={40}
        tint={isDark ? "dark" : "light"}
        style={[base, { backgroundColor: colors.glassBg }, style]}
      >
        {children}
      </BlurView>
    );
  }
  return (
    <View style={[base, { backgroundColor: colors.glassBgFallback }, style]}>
      {children}
    </View>
  );
}
