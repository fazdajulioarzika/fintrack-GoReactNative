import { BlurTargetView } from "expo-blur";
import { LinearGradient } from "expo-linear-gradient";
import React, { createContext, useContext, useRef } from "react";
import { StyleSheet, View } from "react-native";

import { useTheme } from "@/theme/useTheme";

type TargetRef = React.RefObject<View | null>;
const BlurTargetContext = createContext<TargetRef | null>(null);

// Dipakai GlassCard untuk mengetahui bagian mana yang harus diburamkan (diperlukan di Android).
export function useBlurTarget() {
  return useContext(BlurTargetContext);
}

// Latar gradasi + lingkaran warna lembut. Seluruh latar ini adalah "target" blur untuk kartu kaca.
export function GradientBackground({
  children,
}: {
  children: React.ReactNode;
}) {
  const { colors } = useTheme();
  const targetRef = useRef<View | null>(null);

  return (
    <BlurTargetContext.Provider value={targetRef}>
      <View style={styles.fill}>
        <BlurTargetView
          ref={targetRef}
          style={StyleSheet.absoluteFill}
          pointerEvents="none"
        >
          <LinearGradient
            colors={colors.bgGradient}
            style={StyleSheet.absoluteFill}
          />
          <View
            style={[
              styles.blob,
              styles.blobTop,
              { backgroundColor: colors.blobA },
            ]}
          />
          <View
            style={[
              styles.blob,
              styles.blobMid,
              { backgroundColor: colors.blobC },
            ]}
          />
          <View
            style={[
              styles.blob,
              styles.blobBottom,
              { backgroundColor: colors.blobB },
            ]}
          />
        </BlurTargetView>
        {children}
      </View>
    </BlurTargetContext.Provider>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  blob: { position: "absolute", width: 300, height: 300, borderRadius: 150 },
  blobTop: { top: -90, right: -80 },
  blobMid: { top: "38%", left: -130 },
  blobBottom: { bottom: -100, left: -90 },
});
