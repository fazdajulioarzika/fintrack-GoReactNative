import { BlurView } from "expo-blur";
import React from "react";
import { Platform, StyleProp, View, ViewStyle } from "react-native";

import { useBlurTarget } from "@/components/ui/GradientBackground";
import { useTheme } from "@/theme/useTheme";

interface Props {
  children: React.ReactNode;
  style?: StyleProp<ViewStyle>;
  solid?: boolean; // latar solid untuk formulir (PRD 8.3)
  blur?: boolean; // blur sungguhan, pakai hanya untuk kartu utama
}

export function GlassCard({
  children,
  style,
  solid = false,
  blur = false,
}: Props) {
  const { colors, isDark } = useTheme();
  const blurTarget = useBlurTarget();
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

  // Web memakai fallback semi-transparan; iOS dan Android memakai blur asli
  if (blur && Platform.OS !== "web") {
    return (
      <BlurView
        intensity={18}
        tint={isDark ? "dark" : "light"}
        blurTarget={blurTarget ?? undefined}
        blurMethod="dimezisBlurView"
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
