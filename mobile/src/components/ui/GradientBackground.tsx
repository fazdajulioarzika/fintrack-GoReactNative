import { LinearGradient } from "expo-linear-gradient";
import React from "react";
import { StyleSheet, View } from "react-native";

import { useTheme } from "@/theme/useTheme";

// Latar gradasi + dua lingkaran warna lembut agar efek kaca pada kartu terlihat.
export function GradientBackground({
  children,
}: {
  children: React.ReactNode;
}) {
  const { colors } = useTheme();
  return (
    <LinearGradient colors={colors.bgGradient} style={styles.fill}>
      <View
        pointerEvents="none"
        style={[styles.blob, styles.blobTop, { backgroundColor: colors.blobA }]}
      />
      <View
        pointerEvents="none"
        style={[
          styles.blob,
          styles.blobBottom,
          { backgroundColor: colors.blobB },
        ]}
      />
      {children}
    </LinearGradient>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  blob: { position: "absolute", width: 280, height: 280, borderRadius: 140 },
  blobTop: { top: -90, right: -70 },
  blobBottom: { bottom: -100, left: -90 },
});
