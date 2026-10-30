import React from "react";
import { StyleSheet, View } from "react-native";

import { useTheme } from "@/theme/useTheme";

export function ProgressBar({
  percent,
  color,
}: {
  percent: number;
  color: string;
}) {
  const { isDark } = useTheme();
  const width = Math.max(0, Math.min(100, percent));
  return (
    <View
      style={[
        styles.track,
        {
          backgroundColor: isDark
            ? "rgba(255,255,255,0.12)"
            : "rgba(15,23,42,0.08)",
        },
      ]}
    >
      <View
        style={[styles.fill, { width: `${width}%`, backgroundColor: color }]}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  track: { height: 8, borderRadius: 4, overflow: "hidden" },
  fill: { height: 8, borderRadius: 4 },
});
