import React from "react";
import { Pressable, StyleSheet, Text } from "react-native";

import { onColor } from "@/theme/contrast";
import { useTheme } from "@/theme/useTheme";

interface Props {
  label: string;
  selected?: boolean;
  onPress: () => void;
  tint?: string;
  children?: React.ReactNode; // ikon opsional di kiri label
}

export function Chip({
  label,
  selected = false,
  onPress,
  tint,
  children,
}: Props) {
  const { colors } = useTheme();
  const active = tint ?? colors.primaryStrong;
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityState={{ selected }}
      style={[
        styles.chip,
        {
          backgroundColor: selected ? active : colors.inputBg,
          borderColor: selected ? active : colors.inputBorder,
        },
      ]}
    >
      {children}
      <Text
        style={[
          styles.text,
          { color: selected ? onColor(active) : colors.text },
        ]}
      >
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    paddingHorizontal: 14,
    paddingVertical: 9,
    borderRadius: 999,
    borderWidth: 1,
  },
  text: { fontSize: 14, fontWeight: "600" },
});
