import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";

import { currentMonth, monthLabel, shiftMonth } from "@/lib/date";
import { useTheme } from "@/theme/useTheme";

export function MonthSwitcher({
  month,
  onChange,
}: {
  month: string;
  onChange: (m: string) => void;
}) {
  const { colors } = useTheme();
  const isCurrent = month >= currentMonth();
  return (
    <View style={styles.row}>
      <Pressable
        onPress={() => onChange(shiftMonth(month, -1))}
        hitSlop={12}
        accessibilityLabel="Bulan sebelumnya"
      >
        <Ionicons name="chevron-back" size={22} color={colors.text} />
      </Pressable>
      <Text style={[styles.text, { color: colors.text }]}>
        {monthLabel(month)}
      </Text>
      {/* Anggaran boleh dibuat sampai 1 bulan ke depan, jadi batas maju = bulan depan */}
      <Pressable
        onPress={() => onChange(shiftMonth(month, 1))}
        disabled={month > currentMonth()}
        hitSlop={12}
        accessibilityLabel="Bulan berikutnya"
      >
        <Ionicons
          name="chevron-forward"
          size={22}
          color={colors.text}
          style={{ opacity: month > currentMonth() ? 0.3 : 1 }}
        />
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: 18,
    marginVertical: 12,
  },
  text: { fontSize: 16, fontWeight: "700", minWidth: 140, textAlign: "center" },
});
