import React from "react";
import { StyleSheet, Text, TextInput, View } from "react-native";

import { useTheme } from "@/theme/useTheme";

export const MAX_AMOUNT = 999_999_999_999;

export function sanitizeDigits(text: string): string {
  return text.replace(/\D/g, "").replace(/^0+/, "").slice(0, 12);
}

const group = (d: string) => d.replace(/\B(?=(\d{3})+(?!\d))/g, ".");

interface Props {
  digits: string;
  onChangeDigits: (d: string) => void;
  accent: string;
  label?: string;
}

export function AmountInput({
  digits,
  onChangeDigits,
  accent,
  label = "Nominal",
}: Props) {
  const { colors } = useTheme();
  return (
    <View style={{ marginBottom: 16 }}>
      <Text style={[styles.label, { color: colors.text }]}>{label}</Text>
      <View
        style={[
          styles.box,
          { backgroundColor: colors.inputBg, borderColor: colors.inputBorder },
        ]}
      >
        <Text style={[styles.rp, { color: accent }]}>Rp</Text>
        <TextInput
          value={group(digits)}
          onChangeText={(t) => onChangeDigits(sanitizeDigits(t))}
          placeholder="0"
          placeholderTextColor={colors.textMuted}
          keyboardType="number-pad"
          accessibilityLabel={label}
          style={[styles.input, { color: colors.text }]}
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  label: { fontSize: 14, fontWeight: "600", marginBottom: 8 },
  box: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    borderWidth: 1,
    borderRadius: 14,
    paddingHorizontal: 14,
    height: 62,
  },
  rp: { fontSize: 20, fontWeight: "800" },
  input: { flex: 1, fontSize: 26, fontWeight: "800" },
});
