import { Ionicons } from "@expo/vector-icons";
import { useQuery } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import React, { useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { AppHeader } from "@/components/AppHeader";
import { MonthSwitcher } from "@/components/MonthSwitcher";
import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { GlassCard } from "@/components/ui/GlassCard";
import { GradientBackground } from "@/components/ui/GradientBackground";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { api, ApiError } from "@/lib/api";
import { budgetColor, budgetLabel } from "@/lib/budget";
import { currentMonth } from "@/lib/date";
import { formatIDR } from "@/lib/format";
import type { BudgetList, GoalList, IconName } from "@/lib/types";
import { useTheme } from "@/theme/useTheme";

type Tab = "budget" | "savings";

export default function BudgetsScreen() {
  const { colors } = useTheme();
  const router = useRouter();
  const [tab, setTab] = useState<Tab>("budget");
  const [month, setMonth] = useState(currentMonth());
  const [refreshing, setRefreshing] = useState(false);

  const budgets = useQuery({
    queryKey: ["budgets", month],
    queryFn: () => api.get<BudgetList>(`/budgets?month=${month}`),
  });
  const goals = useQuery({
    queryKey: ["savings", "goals"],
    queryFn: () => api.get<GoalList>("/savings-goals"),
  });

  const active = tab === "budget" ? budgets : goals;

  const onRefresh = async () => {
    setRefreshing(true);
    await active.refetch();
    setRefreshing(false);
  };

  return (
    <GradientBackground>
      <View style={styles.fill}>
        <AppHeader title="Anggaran" />
        <ScrollView
          contentContainerStyle={styles.scroll}
          refreshControl={
            <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
          }
        >
          <View style={styles.tabs}>
            <Chip
              label="Anggaran"
              selected={tab === "budget"}
              onPress={() => setTab("budget")}
            />
            <Chip
              label="Target tabungan"
              selected={tab === "savings"}
              onPress={() => setTab("savings")}
            />
          </View>

          {active.isLoading && <ActivityIndicator style={{ marginTop: 30 }} />}
          {active.error && (
            <View style={{ marginTop: 20 }}>
              <Text style={{ color: colors.danger, marginBottom: 10 }}>
                {active.error instanceof ApiError
                  ? active.error.message
                  : "Gagal memuat data"}
              </Text>
              <Button
                title="Coba lagi"
                variant="outline"
                onPress={() => active.refetch()}
              />
            </View>
          )}

          {tab === "budget" && (
            <>
              <MonthSwitcher month={month} onChange={setMonth} />

              {budgets.data && (
                <>
                  <GlassCard style={styles.card}>
                    <Text style={[styles.small, { color: colors.textMuted }]}>
                      Total terpakai
                    </Text>
                    <Text style={[styles.big, { color: colors.text }]}>
                      {formatIDR(budgets.data.total_spent)}
                      <Text style={[styles.small, { color: colors.textMuted }]}>
                        {" "}
                        dari {formatIDR(budgets.data.total_limit)}
                      </Text>
                    </Text>
                  </GlassCard>

                  {budgets.data.data.length === 0 && (
                    <Text
                      style={{
                        color: colors.textMuted,
                        textAlign: "center",
                        marginVertical: 24,
                      }}
                    >
                      Belum ada anggaran untuk bulan ini.
                    </Text>
                  )}

                  {budgets.data.data.map((b) => {
                    const color = budgetColor(b.status, colors);
                    const left = b.limit_amount - b.spent_amount;
                    return (
                      <Pressable
                        key={b.id}
                        onPress={() =>
                          router.push({
                            pathname: "/budget-form",
                            params: { id: b.id, month },
                          })
                        }
                        accessibilityRole="button"
                      >
                        <GlassCard style={styles.card}>
                          <View style={styles.rowBetween}>
                            <View style={styles.nameRow}>
                              <Ionicons
                                name={
                                  (b.category?.icon || "pricetag") as IconName
                                }
                                size={18}
                                color={colors.textMuted}
                              />
                              <Text
                                numberOfLines={1}
                                style={[styles.name, { color: colors.text }]}
                              >
                                {b.category?.name ?? "Kategori"}
                              </Text>
                            </View>
                            <Text style={{ color, fontWeight: "800" }}>
                              {b.usage_percent}%
                            </Text>
                          </View>
                          <View style={{ marginVertical: 10 }}>
                            <ProgressBar
                              percent={b.usage_percent}
                              color={color}
                            />
                          </View>
                          <View style={styles.rowBetween}>
                            <Text
                              style={[
                                styles.small,
                                { color: colors.textMuted },
                              ]}
                            >
                              {formatIDR(b.spent_amount)} dari{" "}
                              {formatIDR(b.limit_amount)}
                            </Text>
                            <Text
                              style={[
                                styles.small,
                                {
                                  color:
                                    left < 0
                                      ? colors.expense
                                      : colors.textMuted,
                                },
                              ]}
                            >
                              {left < 0
                                ? `Lebih ${formatIDR(-left)}`
                                : `Sisa ${formatIDR(left)}`}
                            </Text>
                          </View>
                          {!!budgetLabel(b.status) && (
                            <Text
                              style={{
                                color,
                                fontSize: 12,
                                fontWeight: "700",
                                marginTop: 6,
                              }}
                            >
                              {budgetLabel(b.status)}
                            </Text>
                          )}
                        </GlassCard>
                      </Pressable>
                    );
                  })}

                  <Button
                    title="Tambah anggaran"
                    onPress={() =>
                      router.push({
                        pathname: "/budget-form",
                        params: { month },
                      })
                    }
                  />
                </>
              )}
            </>
          )}

          {tab === "savings" && goals.data && (
            <>
              <GlassCard style={[styles.card, { marginTop: 16 }]}>
                <Text style={[styles.small, { color: colors.textMuted }]}>
                  Total tabungan
                </Text>
                <Text style={[styles.big, { color: colors.text }]}>
                  {formatIDR(goals.data.total_saved)}
                  <Text style={[styles.small, { color: colors.textMuted }]}>
                    {" "}
                    dari {formatIDR(goals.data.total_target)}
                  </Text>
                </Text>
              </GlassCard>

              {goals.data.data.length === 0 && (
                <Text
                  style={{
                    color: colors.textMuted,
                    textAlign: "center",
                    marginVertical: 24,
                  }}
                >
                  Belum ada target tabungan. Buat satu untuk mulai menabung.
                </Text>
              )}

              {goals.data.data.map((g) => {
                const done = g.status === "achieved";
                const color = done ? colors.income : colors.primary;
                return (
                  <Pressable
                    key={g.id}
                    onPress={() =>
                      router.push({
                        pathname: "/goal-detail",
                        params: { id: g.id },
                      })
                    }
                    accessibilityRole="button"
                  >
                    <GlassCard style={styles.card}>
                      <View style={styles.rowBetween}>
                        <View style={styles.nameRow}>
                          <Ionicons
                            name={(g.icon || "wallet") as IconName}
                            size={18}
                            color={color}
                          />
                          <Text
                            numberOfLines={1}
                            style={[styles.name, { color: colors.text }]}
                          >
                            {g.name}
                          </Text>
                        </View>
                        <Text style={{ color, fontWeight: "800" }}>
                          {done ? "Tercapai" : `${g.progress_percent}%`}
                        </Text>
                      </View>
                      <View style={{ marginVertical: 10 }}>
                        <ProgressBar
                          percent={g.progress_percent}
                          color={color}
                        />
                      </View>
                      <Text style={[styles.small, { color: colors.textMuted }]}>
                        {formatIDR(g.saved_amount)} dari{" "}
                        {formatIDR(g.target_amount)}
                      </Text>
                    </GlassCard>
                  </Pressable>
                );
              })}

              <Button
                title="Buat target"
                onPress={() => router.push("/goal-form")}
              />
            </>
          )}
        </ScrollView>
      </View>
    </GradientBackground>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  scroll: {
    padding: 20,
    paddingBottom: 120,
    maxWidth: 560,
    width: "100%",
    alignSelf: "center",
  },
  title: { fontSize: 26, fontWeight: "800", marginBottom: 12 },
  tabs: { flexDirection: "row", gap: 8, marginBottom: 4 },
  card: { padding: 16, marginBottom: 12 },
  rowBetween: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 8,
  },
  nameRow: { flexDirection: "row", alignItems: "center", gap: 8, flex: 1 },
  name: { fontSize: 16, fontWeight: "700", flex: 1 },
  small: { fontSize: 12 },
  big: { fontSize: 24, fontWeight: "800", marginTop: 4 },
});
