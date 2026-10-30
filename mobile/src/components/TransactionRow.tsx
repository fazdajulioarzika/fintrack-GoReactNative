import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";

import { formatDate } from "@/lib/date";
import { formatIDR } from "@/lib/format";
import type { IconName, Transaction } from "@/lib/types";
import { useTheme } from "@/theme/useTheme";

export function TransactionRow({
  tx,
  onPress,
}: {
  tx: Transaction;
  onPress?: () => void;
}) {
  const { colors } = useTheme();
  const isIncome = tx.type === "income";
  const tint = isIncome ? colors.income : colors.expense;

  return (
    <Pressable onPress={onPress} style={styles.row} accessibilityRole="button">
      <View style={[styles.icon, { backgroundColor: tint + "22" }]}>
        <Ionicons
          name={(tx.category?.icon || "pricetag") as IconName}
          size={20}
          color={tint}
        />
      </View>
      <View style={styles.mid}>
        <Text numberOfLines={1} style={[styles.title, { color: colors.text }]}>
          {tx.category?.name ?? "Tanpa kategori"}
        </Text>
        <Text
          numberOfLines={1}
          style={[styles.sub, { color: colors.textMuted }]}
        >
          {formatDate(tx.transaction_date)}
          {tx.note ? ` · ${tx.note}` : ""}
        </Text>
      </View>
      <Text style={[styles.amount, { color: tint }]}>
        {isIncome ? "+" : "-"}
        {formatIDR(tx.amount)}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    paddingVertical: 10,
    gap: 12,
  },
  icon: {
    width: 42,
    height: 42,
    borderRadius: 14,
    alignItems: "center",
    justifyContent: "center",
  },
  mid: { flex: 1 },
  title: { fontSize: 15, fontWeight: "600" },
  sub: { fontSize: 12, marginTop: 2 },
  amount: { fontSize: 15, fontWeight: "700" },
});
