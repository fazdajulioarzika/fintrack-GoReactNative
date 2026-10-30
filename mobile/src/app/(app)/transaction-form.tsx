import { Ionicons } from "@expo/vector-icons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocalSearchParams, useRouter } from "expo-router";
import React, { useEffect, useRef, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
  Pressable,
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
import { TextField } from "@/components/ui/TextField";
import { api, ApiError } from "@/lib/api";
import {
  dateStrToISO,
  isValidDateStr,
  toLocalDateStr,
  todayStr,
} from "@/lib/date";
import { invalidateFinance } from "@/lib/queries";
import type { Category, IconName, Transaction, TxType } from "@/lib/types";
import { useTheme } from "@/theme/useTheme";

const MAX_AMOUNT = 999_999_999_999;
const PAYMENTS = ["Tunai", "Transfer", "E-wallet", "Kartu"];

const groupDigits = (digits: string) =>
  digits.replace(/\B(?=(\d{3})+(?!\d))/g, ".");

export default function TransactionFormScreen() {
  const { id } = useLocalSearchParams<{ id?: string }>();
  const isEdit = !!id;
  const router = useRouter();
  const qc = useQueryClient();
  const { colors } = useTheme();

  const [type, setType] = useState<TxType>("expense");
  const [digits, setDigits] = useState("");
  const [categoryId, setCategoryId] = useState<string | null>(null);
  const [dateText, setDateText] = useState(todayStr());
  const [originalDate, setOriginalDate] = useState("");
  const [payment, setPayment] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  // Kunci anti-duplikasi: dibuat sekali per pembukaan form, jadi tap ganda tidak menggandakan transaksi
  const idemKey = useRef(
    `${Date.now()}-${Math.random().toString(36).slice(2)}`
  ).current;

  const cats = useQuery({
    queryKey: ["categories", type],
    queryFn: () => api.get<{ data: Category[] }>(`/categories?type=${type}`),
    staleTime: 5 * 60_000,
  });

  const existing = useQuery({
    queryKey: ["transaction", id],
    queryFn: () => api.get<{ transaction: Transaction }>(`/transactions/${id}`),
    enabled: isEdit,
  });

  const prefilled = useRef(false);
  useEffect(() => {
    const t = existing.data?.transaction;
    if (t && !prefilled.current) {
      prefilled.current = true;
      const d = toLocalDateStr(t.transaction_date);
      setType(t.type);
      setDigits(String(t.amount));
      setCategoryId(t.category?.id ?? null);
      setDateText(d);
      setOriginalDate(d);
      setPayment(t.payment_method);
      setNote(t.note);
    }
  }, [existing.data]);

  const selectType = (t: TxType) => {
    if (t === type) return;
    setType(t);
    setCategoryId(null); // kategori harus sesuai jenis transaksi
  };

  const onAmountChange = (text: string) => {
    const only = text.replace(/\D/g, "").replace(/^0+/, "").slice(0, 12);
    setDigits(only);
  };

  const save = async () => {
    setError("");
    const amount = parseInt(digits || "0", 10);
    if (!amount || amount <= 0) return setError("Nominal harus lebih dari 0");
    if (amount > MAX_AMOUNT) return setError("Nominal terlalu besar");
    if (!categoryId) return setError("Pilih kategori terlebih dahulu");
    if (!isValidDateStr(dateText))
      return setError(
        "Tanggal tidak valid. Gunakan format TTTT-BB-HH, contoh 2026-10-29"
      );

    const body: Record<string, unknown> = {
      type,
      amount,
      category_id: categoryId,
      payment_method: payment.trim(),
      note: note.trim(),
    };

    setSaving(true);
    try {
      if (isEdit) {
        if (dateText !== originalDate)
          body.transaction_date = dateStrToISO(dateText);
        await api.patch(`/transactions/${id}`, body);
      } else {
        body.transaction_date = dateStrToISO(dateText);
        await api.post("/transactions", body, {
          headers: { "Idempotency-Key": idemKey },
        });
      }
      invalidateFinance(qc);
      qc.invalidateQueries({ queryKey: ["transaction"] });
      router.back();
    } catch (e) {
      setError(
        e instanceof ApiError ? e.message : "Gagal menyimpan. Coba lagi."
      );
    } finally {
      setSaving(false);
    }
  };

  const confirmDelete = () => {
    Alert.alert(
      "Hapus transaksi?",
      "Transaksi ini akan dihapus permanen dan saldo akan dihitung ulang.",
      [
        { text: "Batal", style: "cancel" },
        {
          text: "Hapus",
          style: "destructive",
          onPress: async () => {
            try {
              await api.delete(`/transactions/${id}`);
              invalidateFinance(qc);
              router.back();
            } catch (e) {
              setError(
                e instanceof ApiError
                  ? e.message
                  : "Gagal menghapus. Coba lagi."
              );
            }
          },
        },
      ]
    );
  };

  const accent = type === "income" ? colors.income : colors.expense;

  if (isEdit && existing.isLoading) {
    return (
      <GradientBackground>
        <ActivityIndicator style={{ flex: 1 }} size="large" />
      </GradientBackground>
    );
  }
  if (isEdit && existing.error) {
    return (
      <GradientBackground>
        <SafeAreaView
          style={{ flex: 1, padding: 24, justifyContent: "center" }}
        >
          <Text style={{ color: colors.danger, marginBottom: 12 }}>
            {existing.error instanceof ApiError
              ? existing.error.message
              : "Gagal memuat transaksi"}
          </Text>
          <Button
            title="Kembali"
            variant="outline"
            onPress={() => router.back()}
          />
        </SafeAreaView>
      </GradientBackground>
    );
  }

  return (
    <GradientBackground>
      <SafeAreaView style={styles.fill}>
        <ScreenHeader
          title={isEdit ? "Edit transaksi" : "Tambah transaksi"}
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
              <View style={styles.typeRow}>
                <View style={{ flex: 1 }}>
                  <Chip
                    label="Pengeluaran"
                    selected={type === "expense"}
                    onPress={() => selectType("expense")}
                    tint={colors.expense}
                  />
                </View>
                <View style={{ flex: 1 }}>
                  <Chip
                    label="Pemasukan"
                    selected={type === "income"}
                    onPress={() => selectType("income")}
                    tint={colors.income}
                  />
                </View>
              </View>

              <Text style={[styles.label, { color: colors.text }]}>
                Nominal
              </Text>
              <View
                style={[
                  styles.amountBox,
                  {
                    backgroundColor: colors.inputBg,
                    borderColor: colors.inputBorder,
                  },
                ]}
              >
                <Text style={[styles.rp, { color: accent }]}>Rp</Text>
                <TextField_Amount
                  value={groupDigits(digits)}
                  onChangeText={onAmountChange}
                  color={colors.text}
                />
              </View>

              <Text style={[styles.label, { color: colors.text }]}>
                Kategori
              </Text>
              {cats.isLoading && (
                <ActivityIndicator style={{ marginVertical: 10 }} />
              )}
              {cats.error && (
                <Text style={{ color: colors.danger, marginBottom: 10 }}>
                  Gagal memuat kategori
                </Text>
              )}
              <View style={styles.wrap}>
                {cats.data?.data.map((c) => (
                  <Chip
                    key={c.id}
                    label={c.name}
                    selected={categoryId === c.id}
                    onPress={() => setCategoryId(c.id)}
                    tint={accent}
                  >
                    <Ionicons
                      name={(c.icon || "pricetag") as IconName}
                      size={14}
                      color={categoryId === c.id ? "#FFFFFF" : colors.textMuted}
                    />
                  </Chip>
                ))}
              </View>

              <View style={{ height: 14 }} />
              <TextField
                label="Tanggal (TTTT-BB-HH)"
                value={dateText}
                onChangeText={setDateText}
                placeholder="2026-10-29"
                keyboardType="numbers-and-punctuation"
                maxLength={10}
              />
              <View style={[styles.wrap, { marginTop: -6, marginBottom: 14 }]}>
                <Chip
                  label="Hari ini"
                  selected={dateText === todayStr()}
                  onPress={() => setDateText(todayStr())}
                />
                <Chip
                  label="Kemarin"
                  selected={dateText === todayStr(-1)}
                  onPress={() => setDateText(todayStr(-1))}
                />
              </View>

              <Text style={[styles.label, { color: colors.text }]}>
                Metode pembayaran (opsional)
              </Text>
              <View style={[styles.wrap, { marginBottom: 14 }]}>
                {PAYMENTS.map((p) => (
                  <Chip
                    key={p}
                    label={p}
                    selected={payment === p}
                    onPress={() => setPayment(payment === p ? "" : p)}
                  />
                ))}
              </View>

              <TextField
                label="Catatan (opsional)"
                value={note}
                onChangeText={setNote}
                placeholder="Mis. makan siang"
                maxLength={500}
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
              />
              {isEdit && (
                <View style={{ marginTop: 12 }}>
                  <Button
                    title="Hapus transaksi"
                    variant="outline"
                    onPress={confirmDelete}
                    disabled={saving}
                  />
                </View>
              )}
            </GlassCard>
          </ScrollView>
        </KeyboardAvoidingView>
      </SafeAreaView>
    </GradientBackground>
  );
}

// Input nominal besar, dipisah agar gayanya berbeda dari TextField biasa
import { TextInput } from "react-native";
import { ScreenHeader } from "@/components/ScreenHeader";
function TextField_Amount({
  value,
  onChangeText,
  color,
}: {
  value: string;
  onChangeText: (t: string) => void;
  color: string;
}) {
  return (
    <TextInput
      value={value}
      onChangeText={onChangeText}
      placeholder="0"
      placeholderTextColor="#94A3B8"
      keyboardType="number-pad"
      style={{ flex: 1, fontSize: 26, fontWeight: "800", color }}
      accessibilityLabel="Nominal"
    />
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
    alignItems: "center",
    justifyContent: "space-between",
    marginBottom: 16,
  },
  title: { fontSize: 22, fontWeight: "800" },
  card: { padding: 18 },
  typeRow: { flexDirection: "row", gap: 10, marginBottom: 18 },
  label: { fontSize: 14, fontWeight: "600", marginBottom: 8 },
  amountBox: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    borderWidth: 1,
    borderRadius: 14,
    paddingHorizontal: 14,
    height: 62,
    marginBottom: 18,
  },
  rp: { fontSize: 20, fontWeight: "800" },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  error: { fontSize: 13, marginBottom: 12 },
});
