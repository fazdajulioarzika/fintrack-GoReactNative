import { Ionicons } from "@expo/vector-icons";
import { useQuery } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import React, { useEffect, useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { GlassCard } from "@/components/ui/GlassCard";
import { GradientBackground } from "@/components/ui/GradientBackground";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { TextField } from "@/components/ui/TextField";
import { api, ApiError } from "@/lib/api";
import {
  daysInclusive,
  formatDay,
  isValidDateStr,
  monthLabel,
  todayStr,
} from "@/lib/date";
import { formatIDR } from "@/lib/format";
import type {
  AnalyticsSummary,
  ChangeInfo,
  CompareResult,
  CashflowPoint,
  IconName,
  TrendResult,
} from "@/lib/types";
import { useTheme } from "@/theme/useTheme";
import { ScreenHeader } from "@/components/ScreenHeader";

type RangeKey = "this_month" | "last_month" | "custom";

function TrendChart({
  data,
  granularity,
}: {
  data: CashflowPoint[];
  granularity: "day" | "month";
}) {
  const { colors } = useTheme();
  const [sel, setSel] = useState<number | null>(null);
  useEffect(() => setSel(null), [data]);

  const H = 120;
  const max = Math.max(1, ...data.flatMap((d) => [d.income, d.expense]));
  const hasData = data.some((d) => d.income > 0 || d.expense > 0);
  if (!hasData) {
    return (
      <Text style={{ color: colors.textMuted, marginVertical: 20 }}>
        Belum ada data pada periode ini.
      </Text>
    );
  }

  const h = (v: number) => (v > 0 ? Math.max(3, (v / max) * H) : 0);
  const label = (key: string) =>
    granularity === "month" ? monthLabel(key) : formatDay(key);
  const point = sel !== null ? data[sel] : null;

  return (
    <View>
      <View
        style={{
          flexDirection: "row",
          height: H,
          gap: data.length > 40 ? 1 : 2,
          marginTop: 12,
        }}
      >
        {data.map((d, i) => (
          <Pressable
            key={d.date}
            onPress={() => setSel(sel === i ? null : i)}
            accessibilityLabel={`${label(d.date)}: pemasukan ${formatIDR(
              d.income
            )}, pengeluaran ${formatIDR(d.expense)}`}
            style={{ flex: 1, justifyContent: "flex-end" }}
          >
            <View
              style={{
                flexDirection: "row",
                alignItems: "flex-end",
                gap: 1,
                opacity: sel === null || sel === i ? 1 : 0.35,
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
          </Pressable>
        ))}
      </View>
      <View style={{ marginTop: 10, minHeight: 38 }}>
        {point ? (
          <>
            <Text
              style={{ color: colors.text, fontWeight: "700", fontSize: 13 }}
            >
              {label(point.date)}
            </Text>
            <Text style={{ color: colors.textMuted, fontSize: 12 }}>
              Masuk {formatIDR(point.income)} · Keluar{" "}
              {formatIDR(point.expense)}
            </Text>
          </>
        ) : (
          <Text style={{ color: colors.textMuted, fontSize: 12 }}>
            Ketuk batang untuk melihat detail.
          </Text>
        )}
      </View>
      <View style={{ flexDirection: "row", gap: 16 }}>
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

// goodWhenUp: pemasukan naik = bagus (hijau), pengeluaran naik = buruk (merah)
function ChangeText({
  change,
  goodWhenUp,
}: {
  change: ChangeInfo;
  goodWhenUp: boolean;
}) {
  const { colors } = useTheme();
  if (change.amount === 0) {
    return (
      <Text style={{ color: colors.textMuted, fontSize: 12 }}>
        Tidak berubah
      </Text>
    );
  }
  const up = change.amount > 0;
  const good = up === goodWhenUp;
  const color = good ? colors.income : colors.expense;
  return (
    <Text style={{ color, fontSize: 12, fontWeight: "700" }}>
      {up ? "▲" : "▼"} {formatIDR(Math.abs(change.amount))}
      {change.percent !== null ? ` (${Math.abs(change.percent)}%)` : ""}
    </Text>
  );
}

export default function AnalyticsScreen() {
  const { colors } = useTheme();
  const router = useRouter();

  const [range, setRange] = useState<RangeKey>("this_month");
  const [startText, setStartText] = useState("");
  const [endText, setEndText] = useState("");
  const [applied, setApplied] = useState<{ start: string; end: string } | null>(
    null
  );
  const [customError, setCustomError] = useState("");
  const [refreshing, setRefreshing] = useState(false);

  const selectRange = (r: RangeKey) => {
    setRange(r);
    if (r === "custom" && !startText && !endText) {
      setStartText(todayStr(-29));
      setEndText(todayStr());
    }
  };

  const applyCustom = () => {
    setCustomError("");
    const s = startText.trim();
    const e = endText.trim();
    if (!isValidDateStr(s) || !isValidDateStr(e))
      return setCustomError("Tanggal tidak valid. Gunakan format TTTT-BB-HH");
    if (e < s)
      return setCustomError("Tanggal akhir tidak boleh sebelum tanggal awal");
    if (daysInclusive(s, e) > 366)
      return setCustomError("Rentang tanggal maksimal 366 hari");
    setApplied({ start: s, end: e });
  };

  const qs =
    range === "custom"
      ? applied
        ? `range=custom&start_date=${applied.start}&end_date=${applied.end}`
        : null
      : `range=${range}`;
  // Rentang panjang ditampilkan per bulan agar grafik tetap terbaca
  const granularity: "day" | "month" =
    range === "custom" &&
    applied &&
    daysInclusive(applied.start, applied.end) > 62
      ? "month"
      : "day";
  const enabled = qs !== null;

  const summary = useQuery({
    queryKey: ["analytics", "summary", qs],
    queryFn: () => api.get<AnalyticsSummary>(`/analytics/summary?${qs}`),
    enabled,
  });
  const trend = useQuery({
    queryKey: ["analytics", "trend", qs, granularity],
    queryFn: () =>
      api.get<TrendResult>(`/analytics/trend?${qs}&granularity=${granularity}`),
    enabled,
  });
  const compare = useQuery({
    queryKey: ["analytics", "compare", qs],
    queryFn: () => api.get<CompareResult>(`/analytics/compare?${qs}`),
    enabled,
  });

  const anyError = summary.error || trend.error || compare.error;
  const onRefresh = async () => {
    if (!enabled) return;
    setRefreshing(true);
    await Promise.allSettled([
      summary.refetch(),
      trend.refetch(),
      compare.refetch(),
    ]);
    setRefreshing(false);
  };

  const s = summary.data;

  return (
    <GradientBackground>
      <SafeAreaView style={styles.fill}>
        <ScreenHeader title="Analisis" onBack={() => router.back()} />
        <ScrollView
          contentContainerStyle={styles.scroll}
          refreshControl={
            <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
          }
        >
          <View style={styles.chips}>
            <Chip
              label="Bulan ini"
              selected={range === "this_month"}
              onPress={() => selectRange("this_month")}
            />
            <Chip
              label="Bulan lalu"
              selected={range === "last_month"}
              onPress={() => selectRange("last_month")}
            />
            <Chip
              label="Kustom"
              selected={range === "custom"}
              onPress={() => selectRange("custom")}
            />
          </View>

          {range === "custom" && (
            <GlassCard solid style={styles.card}>
              <View style={styles.dateRow}>
                <View style={{ flex: 1 }}>
                  <TextField
                    label="Dari"
                    value={startText}
                    onChangeText={setStartText}
                    placeholder="2026-10-01"
                    keyboardType="numbers-and-punctuation"
                    maxLength={10}
                  />
                </View>
                <View style={{ flex: 1 }}>
                  <TextField
                    label="Sampai"
                    value={endText}
                    onChangeText={setEndText}
                    placeholder="2026-10-31"
                    keyboardType="numbers-and-punctuation"
                    maxLength={10}
                  />
                </View>
              </View>
              {!!customError && (
                <Text
                  style={{
                    color: colors.danger,
                    fontSize: 13,
                    marginBottom: 10,
                  }}
                >
                  {customError}
                </Text>
              )}
              <Button title="Terapkan" onPress={applyCustom} />
            </GlassCard>
          )}

          {!enabled && (
            <Text
              style={{
                color: colors.textMuted,
                textAlign: "center",
                marginTop: 20,
              }}
            >
              Pilih rentang tanggal lalu tekan Terapkan.
            </Text>
          )}

          {enabled &&
            (summary.isLoading || trend.isLoading || compare.isLoading) && (
              <ActivityIndicator style={{ marginTop: 30 }} />
            )}

          {enabled && !!anyError && (
            <View style={{ marginTop: 16 }}>
              <Text style={{ color: colors.danger, marginBottom: 10 }}>
                {anyError instanceof ApiError
                  ? anyError.message
                  : "Gagal memuat data"}
              </Text>
              <Button title="Coba lagi" variant="outline" onPress={onRefresh} />
            </View>
          )}

          {s && (
            <>
              <Text
                style={{
                  color: colors.textMuted,
                  textAlign: "center",
                  marginBottom: 12,
                  fontSize: 12,
                }}
              >
                {formatDay(s.period.start_date)} –{" "}
                {formatDay(s.period.end_date)}
              </Text>

              <GlassCard style={styles.card}>
                <Text style={[styles.small, { color: colors.textMuted }]}>
                  Selisih arus kas
                </Text>
                <Text
                  style={[
                    styles.big,
                    { color: s.net < 0 ? colors.expense : colors.text },
                  ]}
                >
                  {formatIDR(s.net)}
                </Text>
                <View style={styles.row}>
                  <View style={styles.col}>
                    <Text style={[styles.small, { color: colors.textMuted }]}>
                      Pemasukan
                    </Text>
                    <Text style={[styles.amount, { color: colors.income }]}>
                      {formatIDR(s.total_income)}
                    </Text>
                  </View>
                  <View style={styles.col}>
                    <Text style={[styles.small, { color: colors.textMuted }]}>
                      Pengeluaran
                    </Text>
                    <Text style={[styles.amount, { color: colors.expense }]}>
                      {formatIDR(s.total_expense)}
                    </Text>
                  </View>
                </View>
                <Text
                  style={[
                    styles.small,
                    { color: colors.textMuted, marginTop: 10 },
                  ]}
                >
                  {s.transaction_count} transaksi
                </Text>
              </GlassCard>

              {s.transaction_count === 0 && (
                <Text
                  style={{
                    color: colors.textMuted,
                    textAlign: "center",
                    marginBottom: 16,
                  }}
                >
                  Belum ada transaksi pada periode ini.
                </Text>
              )}
            </>
          )}

          {compare.data && (
            <GlassCard style={styles.card}>
              <Text style={[styles.section, { color: colors.text }]}>
                Dibanding periode sebelumnya
              </Text>
              <Text
                style={[
                  styles.small,
                  { color: colors.textMuted, marginBottom: 6 },
                ]}
              >
                {formatDay(compare.data.previous.period.start_date)} –{" "}
                {formatDay(compare.data.previous.period.end_date)}
              </Text>
              {(
                [
                  [
                    "Pemasukan",
                    compare.data.current.income,
                    compare.data.previous.income,
                    compare.data.change.income,
                    true,
                  ],
                  [
                    "Pengeluaran",
                    compare.data.current.expense,
                    compare.data.previous.expense,
                    compare.data.change.expense,
                    false,
                  ],
                  [
                    "Selisih",
                    compare.data.current.net,
                    compare.data.previous.net,
                    compare.data.change.net,
                    true,
                  ],
                ] as const
              ).map(([name, cur, prev, ch, goodUp]) => (
                <View key={name} style={styles.compareRow}>
                  <View style={{ flex: 1 }}>
                    <Text style={{ color: colors.text, fontWeight: "600" }}>
                      {name}
                    </Text>
                    <Text style={[styles.small, { color: colors.textMuted }]}>
                      Sebelumnya {formatIDR(prev)}
                    </Text>
                  </View>
                  <View style={{ alignItems: "flex-end" }}>
                    <Text style={{ color: colors.text, fontWeight: "700" }}>
                      {formatIDR(cur)}
                    </Text>
                    <ChangeText change={ch} goodWhenUp={goodUp} />
                  </View>
                </View>
              ))}
              {compare.data.change.expense.percent === null && (
                <Text
                  style={[
                    styles.small,
                    { color: colors.textMuted, marginTop: 8 },
                  ]}
                >
                  Persentase tidak tersedia jika periode sebelumnya bernilai 0.
                </Text>
              )}
            </GlassCard>
          )}

          {trend.data && (
            <GlassCard style={styles.card}>
              <Text style={[styles.section, { color: colors.text }]}>
                Tren {trend.data.granularity === "month" ? "bulanan" : "harian"}
              </Text>
              <TrendChart
                data={trend.data.data}
                granularity={trend.data.granularity}
              />
            </GlassCard>
          )}

          {s && (
            <GlassCard style={styles.card}>
              <Text style={[styles.section, { color: colors.text }]}>
                Pengeluaran per kategori
              </Text>
              {s.expense_by_category.length === 0 && (
                <Text style={{ color: colors.textMuted, marginTop: 8 }}>
                  Belum ada pengeluaran pada periode ini.
                </Text>
              )}
              {s.expense_by_category.map((c) => (
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
                        style={{
                          color: colors.text,
                          fontWeight: "600",
                          flex: 1,
                        }}
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
          )}
        </ScrollView>
      </SafeAreaView>
    </GradientBackground>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  scroll: {
    padding: 20,
    paddingBottom: 60,
    maxWidth: 560,
    width: "100%",
    alignSelf: "center",
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    marginBottom: 14,
  },
  title: { fontSize: 22, fontWeight: "800" },
  chips: { flexDirection: "row", gap: 8, marginBottom: 14 },
  card: { padding: 16, marginBottom: 14 },
  dateRow: { flexDirection: "row", gap: 10 },
  small: { fontSize: 12 },
  big: { fontSize: 28, fontWeight: "800", marginTop: 4 },
  row: { flexDirection: "row", gap: 12, marginTop: 12 },
  col: { flex: 1 },
  amount: { fontSize: 16, fontWeight: "700", marginTop: 4 },
  section: { fontSize: 16, fontWeight: "700" },
  compareRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    paddingVertical: 8,
  },
  catHead: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    marginBottom: 6,
    gap: 8,
  },
});
