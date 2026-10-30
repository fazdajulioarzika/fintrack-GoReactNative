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
import { GlassCard } from "@/components/ui/GlassCard";
import { GradientBackground } from "@/components/ui/GradientBackground";
import { TextField } from "@/components/ui/TextField";
import { api, ApiError } from "@/lib/api";
import { confirmAction } from "@/lib/confirm";
import { dateStrToISO, isValidDateStr, toLocalDateStr } from "@/lib/date";
import type { IconName, SavingsGoal } from "@/lib/types";
import { useTheme } from "@/theme/useTheme";
import { ScreenHeader } from "@/components/ScreenHeader";

const ICONS: IconName[] = [
  "wallet",
  "shield-checkmark",
  "airplane",
  "school",
  "home",
  "car",
  "phone-portrait",
  "gift",
  "heart",
  "bicycle",
];

export default function GoalFormScreen() {
  const { id } = useLocalSearchParams<{ id?: string }>();
  const isEdit = !!id;
  const router = useRouter();
  const qc = useQueryClient();
  const { colors } = useTheme();

  const [name, setName] = useState("");
  const [digits, setDigits] = useState("");
  const [dateText, setDateText] = useState("");
  const [hadDate, setHadDate] = useState(false);
  const [icon, setIcon] = useState<string>("wallet");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const existing = useQuery({
    queryKey: ["savings", "goal", id],
    queryFn: () => api.get<{ goal: SavingsGoal }>(`/savings-goals/${id}`),
    enabled: isEdit,
  });

  const prefilled = useRef(false);
  useEffect(() => {
    const g = existing.data?.goal;
    if (g && !prefilled.current) {
      prefilled.current = true;
      setName(g.name);
      setDigits(String(g.target_amount));
      setIcon(g.icon || "wallet");
      setHadDate(!!g.target_date);
      setDateText(g.target_date ? toLocalDateStr(g.target_date) : "");
    }
  }, [existing.data]);

  const save = async () => {
    setError("");
    const amount = parseInt(digits || "0", 10);
    if (name.trim().length < 1) return setError("Nama target wajib diisi");
    if (!amount || amount <= 0)
      return setError("Nominal target harus lebih dari 0");
    if (amount > MAX_AMOUNT) return setError("Nominal terlalu besar");
    const hasDate = dateText.trim() !== "";
    if (hasDate && !isValidDateStr(dateText.trim())) {
      return setError(
        "Tanggal tidak valid. Gunakan format TTTT-BB-HH, contoh 2027-06-30"
      );
    }

    const body: Record<string, unknown> = {
      name: name.trim(),
      target_amount: amount,
      icon,
    };
    if (hasDate) body.target_date = dateStrToISO(dateText.trim());
    if (isEdit && !hasDate && hadDate) body.clear_target_date = true;

    setSaving(true);
    try {
      if (isEdit) await api.patch(`/savings-goals/${id}`, body);
      else await api.post("/savings-goals", body);
      qc.invalidateQueries({ queryKey: ["savings"] });
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
      "Hapus target?",
      "Target beserta seluruh riwayat setoran dan penarikannya akan dihapus permanen."
    );
    if (!ok) return;
    try {
      await api.delete(`/savings-goals/${id}`);
      qc.invalidateQueries({ queryKey: ["savings"] });
      router.dismissAll(); // tutup form sekaligus halaman detail yang sudah tidak berlaku
    } catch (e) {
      setError(
        e instanceof ApiError ? e.message : "Gagal menghapus. Coba lagi."
      );
    }
  };

  if (isEdit && existing.isLoading) {
    return (
      <GradientBackground>
        <ActivityIndicator style={{ flex: 1 }} size="large" />
      </GradientBackground>
    );
  }

  return (
    <GradientBackground>
      <SafeAreaView style={styles.fill}>
        <ScreenHeader
          title={isEdit ? "Ubah target" : "Buat target"}
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
              {existing.error && (
                <Text style={{ color: colors.danger, marginBottom: 12 }}>
                  {existing.error instanceof ApiError
                    ? existing.error.message
                    : "Gagal memuat target"}
                </Text>
              )}

              <TextField
                label="Nama target"
                value={name}
                onChangeText={setName}
                placeholder="Mis. Dana darurat"
                maxLength={100}
              />
              <AmountInput
                label="Nominal target"
                digits={digits}
                onChangeDigits={setDigits}
                accent={colors.primary}
              />
              <TextField
                label="Target tanggal (opsional, TTTT-BB-HH)"
                value={dateText}
                onChangeText={setDateText}
                placeholder="2027-06-30"
                keyboardType="numbers-and-punctuation"
                maxLength={10}
              />

              <Text style={[styles.label, { color: colors.text }]}>Ikon</Text>
              <View style={styles.wrap}>
                {ICONS.map((n) => {
                  const selected = icon === n;
                  return (
                    <Pressable
                      key={n}
                      onPress={() => setIcon(n)}
                      accessibilityLabel={`Ikon ${n}`}
                      accessibilityState={{ selected }}
                      style={[
                        styles.iconBox,
                        {
                          backgroundColor: selected
                            ? colors.primary
                            : colors.inputBg,
                          borderColor: selected
                            ? colors.primary
                            : colors.inputBorder,
                        },
                      ]}
                    >
                      <Ionicons
                        name={n}
                        size={22}
                        color={selected ? "#FFFFFF" : colors.textMuted}
                      />
                    </Pressable>
                  );
                })}
              </View>

              {!!error && (
                <Text style={[styles.error, { color: colors.danger }]}>
                  {error}
                </Text>
              )}

              <View style={{ marginTop: 16 }}>
                <Button
                  title={isEdit ? "Simpan perubahan" : "Simpan"}
                  onPress={save}
                  loading={saving}
                />
              </View>
              {isEdit && (
                <View style={{ marginTop: 12 }}>
                  <Button
                    title="Hapus target"
                    variant="outline"
                    onPress={remove}
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
  label: { fontSize: 14, fontWeight: "600", marginBottom: 8 },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  iconBox: {
    width: 46,
    height: 46,
    borderRadius: 14,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
  },
  error: { fontSize: 13, marginTop: 12 },
});
