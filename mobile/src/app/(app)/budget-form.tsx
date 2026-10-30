import { Ionicons } from "@expo/vector-icons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocalSearchParams, useRouter } from "expo-router";
import React, { useEffect, useRef, useState } from "react";
import {
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { AmountInput, MAX_AMOUNT } from "@/components/ui/AmountInput";
import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { GlassCard } from "@/components/ui/GlassCard";
import { GradientBackground } from "@/components/ui/GradientBackground";
import { api, ApiError } from "@/lib/api";
import { confirmAction } from "@/lib/confirm";
import { currentMonth, monthLabel } from "@/lib/date";
import { invalidateFinance } from "@/lib/queries";
import type { BudgetList, Category, IconName } from "@/lib/types";
import { useTheme } from "@/theme/useTheme";
import { ScreenHeader } from "@/components/ScreenHeader";

export default function BudgetFormScreen() {
  const { id, month: monthParam } = useLocalSearchParams<{
    id?: string;
    month?: string;
  }>();
  const month = monthParam || currentMonth();
  const isEdit = !!id;
  const router = useRouter();
  const qc = useQueryClient();
  const { colors } = useTheme();

  const [categoryId, setCategoryId] = useState<string | null>(null);
  const [digits, setDigits] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const budgets = useQuery({
    queryKey: ["budgets", month],
    queryFn: () => api.get<BudgetList>(`/budgets?month=${month}`),
  });
  const cats = useQuery({
    queryKey: ["categories", "expense"],
    queryFn: () => api.get<{ data: Category[] }>("/categories?type=expense"),
    staleTime: 5 * 60_000,
    enabled: !isEdit,
  });

  const editing = budgets.data?.data.find((b) => b.id === id);

  const prefilled = useRef(false);
  useEffect(() => {
    if (editing && !prefilled.current) {
      prefilled.current = true;
      setDigits(String(editing.limit_amount));
    }
  }, [editing]);

  // Satu anggaran per kategori per bulan: sembunyikan kategori yang sudah punya anggaran
  const taken = new Set((budgets.data?.data ?? []).map((b) => b.category?.id));
  const available = (cats.data?.data ?? []).filter((c) => !taken.has(c.id));

  const save = async () => {
    setError("");
    const amount = parseInt(digits || "0", 10);
    if (!amount || amount <= 0)
      return setError("Batas anggaran harus lebih dari 0");
    if (amount > MAX_AMOUNT) return setError("Nominal terlalu besar");
    if (!isEdit && !categoryId)
      return setError("Pilih kategori terlebih dahulu");

    setSaving(true);
    try {
      if (isEdit) {
        await api.patch(`/budgets/${id}`, { limit_amount: amount });
      } else {
        await api.post("/budgets", {
          category_id: categoryId,
          period: month,
          limit_amount: amount,
        });
      }
      invalidateFinance(qc);
      router.back();
    } catch (e) {
      setError(
        e instanceof ApiError ? e.message : "Gagal menyimpan. Coba lagi."
      );
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    const ok = await confirmAction(
      "Hapus anggaran?",
      "Anggaran ini akan dihapus. Transaksi Anda tidak terpengaruh."
    );
    if (!ok) return;
    try {
      await api.delete(`/budgets/${id}`);
      invalidateFinance(qc);
      router.back();
    } catch (e) {
      setError(
        e instanceof ApiError ? e.message : "Gagal menghapus. Coba lagi."
      );
    }
  };

  const loading = budgets.isLoading || (!isEdit && cats.isLoading);

  return (
    <GradientBackground>
      <SafeAreaView style={styles.fill}>
        <ScreenHeader
          title={isEdit ? "Ubah anggaran" : "Tambah anggaran"}
          subtitle={monthLabel(month)}
          onClose={() => router.back()}
        />
        <KeyboardAvoidingView
          style={styles.fill}
          behavior={Platform.OS === "ios" ? "padding" : undefined}
        >
          <ScrollView
            contentContainerStyle={styles.scroll}
            keyboardShouldPersistTaps="handled"
          >
            <GlassCard solid style={styles.card}>
              {loading && <ActivityIndicator style={{ marginVertical: 20 }} />}
              {(budgets.error || cats.error) && (
                <Text style={{ color: colors.danger, marginBottom: 12 }}>
                  Gagal memuat data. Tutup lalu coba lagi.
                </Text>
              )}

              {!loading && isEdit && !editing && !budgets.error && (
                <Text style={{ color: colors.danger, marginBottom: 12 }}>
                  Anggaran tidak ditemukan.
                </Text>
              )}

              {!loading && (isEdit ? !!editing : true) && (
                <>
                  {isEdit ? (
                    <View style={{ marginBottom: 16 }}>
                      <Text style={[styles.label, { color: colors.textMuted }]}>
                        Kategori
                      </Text>
                      <Text
                        style={{
                          color: colors.text,
                          fontSize: 18,
                          fontWeight: "700",
                        }}
                      >
                        {editing?.category?.name}
                      </Text>
                    </View>
                  ) : (
                    <>
                      <Text style={[styles.label, { color: colors.text }]}>
                        Kategori pengeluaran
                      </Text>
                      {available.length === 0 ? (
                        <Text
                          style={{ color: colors.textMuted, marginBottom: 16 }}
                        >
                          Semua kategori sudah punya anggaran untuk bulan ini.
                        </Text>
                      ) : (
                        <View style={styles.wrap}>
                          {available.map((c) => (
                            <Chip
                              key={c.id}
                              label={c.name}
                              selected={categoryId === c.id}
                              onPress={() => setCategoryId(c.id)}
                              tint={colors.secondary}
                            >
                              <Ionicons
                                name={(c.icon || "pricetag") as IconName}
                                size={14}
                                color={
                                  categoryId === c.id
                                    ? "#FFFFFF"
                                    : colors.textMuted
                                }
                              />
                            </Chip>
                          ))}
                        </View>
                      )}
                      <View style={{ height: 16 }} />
                    </>
                  )}

                  <AmountInput
                    label="Batas anggaran"
                    digits={digits}
                    onChangeDigits={setDigits}
                    accent={colors.primary}
                  />

                  {!!error && (
                    <Text style={[styles.error, { color: colors.danger }]}>
                      {error}
                    </Text>
                  )}

                  <Button
                    title={isEdit ? "Simpan perubahan" : "Simpan"}
                    onPress={save}
                    loading={saving}
                    disabled={!isEdit && available.length === 0}
                  />
                  {isEdit && (
                    <View style={{ marginTop: 12 }}>
                      <Button
                        title="Hapus anggaran"
                        variant="outline"
                        onPress={remove}
                        disabled={saving}
                      />
                    </View>
                  )}
                </>
              )}
            </GlassCard>
          </ScrollView>
        </KeyboardAvoidingView>
      </SafeAreaView>
    </GradientBackground>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  scroll: {
    padding: 20,
    paddingBottom: 40,
    maxWidth: 560,
    width: "100%",
    alignSelf: "center",
  },
  header: {
    flexDirection: "row",
    alignItems: "flex-start",
    justifyContent: "space-between",
    marginBottom: 16,
  },
  title: { fontSize: 22, fontWeight: "800" },
  card: { padding: 18 },
  label: { fontSize: 14, fontWeight: "600", marginBottom: 8 },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  error: { fontSize: 13, marginBottom: 12 },
});
