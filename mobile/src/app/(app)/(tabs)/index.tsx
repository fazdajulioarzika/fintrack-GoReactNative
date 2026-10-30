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
import { AppHeader, HeaderIconButton } from "@/components/AppHeader";
import { TransactionRow } from "@/components/TransactionRow";
import { Button } from "@/components/ui/Button";
import { GlassCard } from "@/components/ui/GlassCard";
import { GradientBackground } from "@/components/ui/GradientBackground";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { useAuth } from "@/context/AuthContext";
import { api, ApiError } from "@/lib/api";
import { currentMonth, monthLabel, shiftMonth } from "@/lib/date";
import { formatIDR } from "@/lib/format";
import type {
  Budget,
  CashflowPoint,
  CategoryShare,
  IconName,
  Summary,
  TxList,
} from "@/lib/types";
import { useTheme } from "@/theme/useTheme";

function CashflowBars({ data }: { data: CashflowPoint[] }) {
  const { colors } = useTheme();
  const H = 90;
  const max = Math.max(1, ...data.flatMap((d) => [d.income, d.expense]));
  const hasData = data.some((d) => d.income > 0 || d.expense > 0);

  if (!hasData) {
    return (
      <Text style={{ color: colors.textMuted, marginVertical: 20 }}>
        Belum ada arus kas bulan ini.
      </Text>
    );
  }
  const h = (v: number) => (v > 0 ? Math.max(3, (v / max) * H) : 0);
  return (
    <View>
      <View
        style={{
          flexDirection: "row",
          alignItems: "flex-end",
          height: H,
          gap: 2,
          marginTop: 12,
        }}
      >
        {data.map((d) => (
          <View
            key={d.date}
            style={{
              flex: 1,
              flexDirection: "row",
              alignItems: "flex-end",
              gap: 1,
            }}
          >
            <View
              style={{
                flex: 1,
                height: h(d.income),
                backgroundColor: colors.income,
                borderRadius: 2,
              }}
            />
            <View
              style={{
                flex: 1,
                height: h(d.expense),
                backgroundColor: colors.expense,
                borderRadius: 2,
              }}
            />
          </View>
        ))}
      </View>
      <View style={{ flexDirection: "row", gap: 16, marginTop: 10 }}>
        <Legend color={colors.income} label="Pemasukan" />
        <Legend color={colors.expense} label="Pengeluaran" />
      </View>
    </View>
  );
}

function Legend({ color, label }: { color: string; label: string }) {
  const { colors } = useTheme();
  return (
    <View style={{ flexDirection: "row", alignItems: "center", gap: 6 }}>
      <View
        style={{
          width: 10,
          height: 10,
          borderRadius: 5,
          backgroundColor: color,
        }}
      />
      <Text style={{ color: colors.textMuted, fontSize: 12 }}>{label}</Text>
    </View>
  );
}

export default function HomeScreen() {
  const { colors } = useTheme();
  const { user } = useAuth();
  const router = useRouter();
  const [month, setMonth] = useState(currentMonth());
  const [refreshing, setRefreshing] = useState(false);

  const summary = useQuery({
    queryKey: ["dashboard", "summary", month],
    queryFn: () => api.get<Summary>(`/dashboard/summary?month=${month}`),
  });
  const cashflow = useQuery({
    queryKey: ["dashboard", "cashflow", month],
    queryFn: () =>
      api.get<{ data: CashflowPoint[] }>(`/dashboard/cashflow?month=${month}`),
  });
  const cats = useQuery({
    queryKey: ["dashboard", "categories", month],
    queryFn: () =>
      api.get<{ data: CategoryShare[] }>(
        `/dashboard/categories?month=${month}&type=expense`
      ),
  });
  const budgets = useQuery({
    queryKey: ["budgets", month],
    queryFn: () => api.get<{ data: Budget[] }>(`/budgets?month=${month}`),
  });
  const recent = useQuery({
    queryKey: ["transactions", "recent"],
    queryFn: () => api.get<TxList>("/transactions?limit=5"),
  });

  const onRefresh = async () => {
    setRefreshing(true);
    await Promise.allSettled([
      summary.refetch(),
      cashflow.refetch(),
      cats.refetch(),
      budgets.refetch(),
      recent.refetch(),
    ]);
    setRefreshing(false);
  };

  const statusColor = (s: Budget["status"]) =>
    s === "exceeded"
      ? colors.expense
      : s === "warning"
      ? "#F59E0B"
      : colors.primary;

  return (
    <GradientBackground>
      <View style={styles.fill}>
        <AppHeader
          title="Beranda"
          right={
            <HeaderIconButton
              icon="stats-chart"
              label="Analisis"
              onPress={() => router.push("/analytics")}
            />
          }
        />
        <ScrollView
          contentContainerStyle={styles.scroll}
          refreshControl={
            <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
          }
        >
          <Text style={[styles.greeting, { color: colors.textMuted }]}>
            Halo,
          </Text>
          <Text style={[styles.name, { color: colors.text }]}>
            {user?.name}
          </Text>

          <View style={styles.monthRow}>
            <Pressable
              onPress={() => setMonth(shiftMonth(month, -1))}
              hitSlop={12}
              accessibilityLabel="Bulan sebelumnya"
            >
              <Ionicons name="chevron-back" size={22} color={colors.text} />
            </Pressable>
            <Text style={[styles.monthText, { color: colors.text }]}>
              {monthLabel(month)}
            </Text>
            <Pressable
              onPress={() => setMonth(shiftMonth(month, 1))}
              disabled={month >= currentMonth()}
              hitSlop={12}
              accessibilityLabel="Bulan berikutnya"
            >
              <Ionicons
                name="chevron-forward"
                size={22}
                color={month >= currentMonth() ? colors.textMuted : colors.text}
                style={{ opacity: month >= currentMonth() ? 0.35 : 1 }}
              />
            </Pressable>
          </View>

          <GlassCard style={styles.card}>
            <Text style={[styles.label, { color: colors.textMuted }]}>
              Total saldo
            </Text>
            {summary.isLoading && (
              <ActivityIndicator style={{ marginVertical: 14 }} />
            )}
            {summary.error && (
              <View style={{ marginTop: 8 }}>
                <Text style={{ color: colors.danger, marginBottom: 10 }}>
                  {summary.error instanceof ApiError
                    ? summary.error.message
                    : "Gagal memuat data"}
                </Text>
                <Button
                  title="Coba lagi"
                  variant="outline"
                  onPress={() => summary.refetch()}
                />
              </View>
            )}
            {summary.data && (
              <>
                <Text style={[styles.balance, { color: colors.text }]}>
                  {formatIDR(summary.data.balance)}
                </Text>
                <View style={styles.row}>
                  <View style={styles.col}>
                    <Text style={[styles.label, { color: colors.textMuted }]}>
                      Pemasukan
                    </Text>
                    <Text style={[styles.amount, { color: colors.income }]}>
                      {formatIDR(summary.data.month_income)}
                    </Text>
                  </View>
                  <View style={styles.col}>
                    <Text style={[styles.label, { color: colors.textMuted }]}>
                      Pengeluaran
                    </Text>
                    <Text style={[styles.amount, { color: colors.expense }]}>
                      {formatIDR(summary.data.month_expense)}
                    </Text>
                  </View>
                </View>
              </>
            )}
          </GlassCard>

          <GlassCard style={styles.card}>
            <View style={styles.catHead}>
              <Text style={[styles.section, { color: colors.text }]}>
                Arus kas
              </Text>
              <Pressable onPress={() => router.push("/analytics")} hitSlop={8}>
                <Text style={{ color: colors.primary, fontWeight: "700" }}>
                  Analisis
                </Text>
              </Pressable>
            </View>
            {cashflow.isLoading ? (
              <ActivityIndicator style={{ marginVertical: 20 }} />
            ) : (
              <CashflowBars data={cashflow.data?.data ?? []} />
            )}
          </GlassCard>

          <GlassCard style={styles.card}>
            <Text style={[styles.section, { color: colors.text }]}>
              Pengeluaran per kategori
            </Text>
            {cats.isLoading && (
              <ActivityIndicator style={{ marginVertical: 14 }} />
            )}
            {cats.data && cats.data.data.length === 0 && (
              <Text style={{ color: colors.textMuted, marginTop: 8 }}>
                Belum ada pengeluaran bulan ini.
              </Text>
            )}
            {cats.data?.data.slice(0, 5).map((c) => (
              <View key={c.category_id} style={{ marginTop: 12 }}>
                <View style={styles.catHead}>
                  <View
                    style={{
                      flexDirection: "row",
                      alignItems: "center",
                      gap: 8,
                      flex: 1,
                    }}
                  >
                    <Ionicons
                      name={(c.icon || "pricetag") as IconName}
                      size={16}
                      color={colors.textMuted}
                    />
                    <Text
                      numberOfLines={1}
                      style={{ color: colors.text, fontWeight: "600", flex: 1 }}
                    >
                      {c.name}
                    </Text>
                  </View>
                  <Text style={{ color: colors.textMuted, fontSize: 13 }}>
                    {formatIDR(c.total)} · {c.percent}%
                  </Text>
                </View>
                <ProgressBar percent={c.percent} color={colors.secondary} />
              </View>
            ))}
          </GlassCard>

          {budgets.data && budgets.data.data.length > 0 && (
            <GlassCard style={styles.card}>
              <Text style={[styles.section, { color: colors.text }]}>
                Anggaran bulan ini
              </Text>
              {budgets.data.data.slice(0, 3).map((b) => (
                <View key={b.id} style={{ marginTop: 12 }}>
                  <View style={styles.catHead}>
                    <Text
                      numberOfLines={1}
                      style={{ color: colors.text, fontWeight: "600", flex: 1 }}
                    >
                      {b.category?.name ?? "Kategori"}
                    </Text>
                    <Text
                      style={{
                        color: statusColor(b.status),
                        fontSize: 13,
                        fontWeight: "700",
                      }}
                    >
                      {b.usage_percent}%
                    </Text>
                  </View>
                  <ProgressBar
                    percent={b.usage_percent}
                    color={statusColor(b.status)}
                  />
                  <Text
                    style={{
                      color: colors.textMuted,
                      fontSize: 12,
                      marginTop: 4,
                    }}
                  >
                    {formatIDR(b.spent_amount)} dari {formatIDR(b.limit_amount)}
                  </Text>
                </View>
              ))}
            </GlassCard>
          )}

          <GlassCard style={styles.card}>
            <View style={styles.catHead}>
              <Text style={[styles.section, { color: colors.text }]}>
                Transaksi terbaru
              </Text>
              <Pressable
                onPress={() => router.push("/transactions")}
                hitSlop={8}
              >
                <Text style={{ color: colors.primary, fontWeight: "700" }}>
                  Lihat semua
                </Text>
              </Pressable>
            </View>
            {recent.isLoading && (
              <ActivityIndicator style={{ marginVertical: 14 }} />
            )}
            {recent.data && recent.data.data.length === 0 && (
              <Text style={{ color: colors.textMuted, marginTop: 8 }}>
                Belum ada transaksi. Tekan tombol + untuk menambah.
              </Text>
            )}
            {recent.data?.data.map((tx) => (
              <TransactionRow
                key={tx.id}
                tx={tx}
                onPress={() =>
                  router.push({
                    pathname: "/transaction-form",
                    params: { id: tx.id },
                  })
                }
              />
            ))}
          </GlassCard>
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
  greeting: { fontSize: 14 },
  name: { fontSize: 26, fontWeight: "800" },
  monthRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: 18,
    marginVertical: 16,
  },
  monthText: {
    fontSize: 16,
    fontWeight: "700",
    minWidth: 140,
    textAlign: "center",
  },
  card: { padding: 18, marginBottom: 16 },
  label: { fontSize: 12 },
  balance: { fontSize: 32, fontWeight: "800", marginVertical: 8 },
  row: { flexDirection: "row", gap: 12, marginTop: 8 },
  col: { flex: 1 },
  amount: { fontSize: 16, fontWeight: "700", marginTop: 4 },
  section: { fontSize: 16, fontWeight: "700" },
  catHead: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    marginBottom: 6,
    gap: 8,
  },
});
