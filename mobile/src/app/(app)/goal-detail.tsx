import { Ionicons } from "@expo/vector-icons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocalSearchParams, useRouter } from "expo-router";
import React, { useRef, useState } from "react";
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
import { ProgressBar } from "@/components/ui/ProgressBar";
import { TextField } from "@/components/ui/TextField";
import { api, ApiError } from "@/lib/api";
import { formatDate } from "@/lib/date";
import { formatIDR } from "@/lib/format";
import type { Contribution, IconName, SavingsGoal } from "@/lib/types";
import { useTheme } from "@/theme/useTheme";
import { HeaderIconButton } from "@/components/AppHeader";
import { ScreenHeader } from "@/components/ScreenHeader";

type Mode = "deposit" | "withdraw";
const newKey = () => `${Date.now()}-${Math.random().toString(36).slice(2)}`;

export default function GoalDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const qc = useQueryClient();
  const { colors } = useTheme();

  const [mode, setMode] = useState<Mode | null>(null);
  const [digits, setDigits] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const idemKey = useRef(newKey()); // diganti setelah tiap setoran sukses; tap ganda memakai key yang sama

  const goal = useQuery({
    queryKey: ["savings", "goal", id],
    queryFn: () => api.get<{ goal: SavingsGoal }>(`/savings-goals/${id}`),
  });
  const history = useQuery({
    queryKey: ["savings", "contributions", id],
    queryFn: () =>
      api.get<{ data: Contribution[] }>(
        `/savings-goals/${id}/contributions?limit=50`
      ),
  });

  const g = goal.data?.goal;
  const done = g?.status === "achieved";
  const color = done ? colors.income : colors.primary;

  const openMode = (m: Mode) => {
    setMode(m);
    setDigits("");
    setNote("");
    setError("");
  };

  const submit = async () => {
    if (!mode) return;
    setError("");
    const amount = parseInt(digits || "0", 10);
    if (!amount || amount <= 0) return setError("Nominal harus lebih dari 0");
    if (amount > MAX_AMOUNT) return setError("Nominal terlalu besar");
    if (mode === "withdraw" && g && amount > g.saved_amount) {
      return setError(
        `Penarikan melebihi tabungan terkumpul (${formatIDR(g.saved_amount)})`
      );
    }

    setSaving(true);
    try {
      await api.post(
        `/savings-goals/${id}/contributions`,
        { type: mode, amount, note: note.trim() },
        { headers: { "Idempotency-Key": idemKey.current } }
      );
      idemKey.current = newKey();
      qc.invalidateQueries({ queryKey: ["savings"] });
      setMode(null);
    } catch (e) {
      // Key tidak diganti: mengulang request yang sama tidak akan menggandakan setoran
      setError(
        e instanceof ApiError ? e.message : "Gagal menyimpan. Coba lagi."
      );
    } finally {
      setSaving(false);
    }
  };

  return (
    <GradientBackground>
      <SafeAreaView style={styles.fill}>
        <ScreenHeader
          title={g?.name ?? "Target tabungan"}
          onBack={() => router.back()}
          right={
            g ? (
              <HeaderIconButton
                icon="create-outline"
                label="Ubah target"
                onPress={() =>
                  router.push({ pathname: "/goal-form", params: { id } })
                }
              />
            ) : undefined
          }
        />
        <KeyboardAvoidingView
          style={styles.fill}
          behavior={Platform.OS === "ios" ? "padding" : undefined}
        >
          <ScrollView
            contentContainerStyle={styles.scroll}
            keyboardShouldPersistTaps="handled"
          >
            {goal.isLoading && (
              <ActivityIndicator style={{ marginTop: 40 }} size="large" />
            )}
            {goal.error && (
              <View>
                <Text style={{ color: colors.danger, marginBottom: 10 }}>
                  {goal.error instanceof ApiError
                    ? goal.error.message
                    : "Gagal memuat target"}
                </Text>
                <Button
                  title="Kembali"
                  variant="outline"
                  onPress={() => router.back()}
                />
              </View>
            )}

            {g && (
              <>
                <GlassCard style={styles.card}>
                  <View style={styles.nameRow}>
                    <Ionicons
                      name={(g.icon || "wallet") as IconName}
                      size={22}
                      color={color}
                    />
                    <Text style={[styles.name, { color: colors.text }]}>
                      {g.name}
                    </Text>
                  </View>
                  <Text style={[styles.big, { color: colors.text }]}>
                    {formatIDR(g.saved_amount)}
                  </Text>
                  <Text style={{ color: colors.textMuted, marginBottom: 12 }}>
                    dari {formatIDR(g.target_amount)}
                  </Text>
                  <ProgressBar percent={g.progress_percent} color={color} />
                  <View style={styles.rowBetween}>
                    <Text style={{ color, fontWeight: "800", marginTop: 8 }}>
                      {done ? "Target tercapai 🎉" : `${g.progress_percent}%`}
                    </Text>
                    {!done && (
                      <Text
                        style={{
                          color: colors.textMuted,
                          marginTop: 8,
                          fontSize: 12,
                        }}
                      >
                        Sisa {formatIDR(g.remaining_amount)}
                      </Text>
                    )}
                  </View>
                  {g.target_date && (
                    <Text
                      style={{
                        color: colors.textMuted,
                        fontSize: 12,
                        marginTop: 6,
                      }}
                    >
                      Target tanggal: {formatDate(g.target_date)}
                    </Text>
                  )}
                </GlassCard>

                {mode === null ? (
                  <View style={styles.actions}>
                    <View style={{ flex: 1 }}>
                      <Button
                        title="Setor"
                        onPress={() => openMode("deposit")}
                      />
                    </View>
                    <View style={{ flex: 1 }}>
                      <Button
                        title="Tarik"
                        variant="outline"
                        onPress={() => openMode("withdraw")}
                        disabled={g.saved_amount === 0}
                      />
                    </View>
                  </View>
                ) : (
                  <GlassCard solid style={styles.card}>
                    <Text style={[styles.section, { color: colors.text }]}>
                      {mode === "deposit" ? "Setor tabungan" : "Tarik tabungan"}
                    </Text>
                    <AmountInput
                      digits={digits}
                      onChangeDigits={setDigits}
                      accent={
                        mode === "deposit" ? colors.income : colors.expense
                      }
                    />
                    <TextField
                      label="Catatan (opsional)"
                      value={note}
                      onChangeText={setNote}
                      placeholder="Mis. sisa gaji"
                      maxLength={500}
                    />
                    {!!error && (
                      <Text style={[styles.error, { color: colors.danger }]}>
                        {error}
                      </Text>
                    )}
                    <Button title="Simpan" onPress={submit} loading={saving} />
                    <View style={{ marginTop: 10 }}>
                      <Button
                        title="Batal"
                        variant="outline"
                        onPress={() => setMode(null)}
                        disabled={saving}
                      />
                    </View>
                  </GlassCard>
                )}

                <Text
                  style={[
                    styles.section,
                    { color: colors.text, marginTop: 8, marginBottom: 10 },
                  ]}
                >
                  Riwayat
                </Text>
                {history.isLoading && <ActivityIndicator />}
                {history.data && history.data.data.length === 0 && (
                  <Text style={{ color: colors.textMuted }}>
                    Belum ada setoran. Tekan Setor untuk memulai.
                  </Text>
                )}
                {history.data?.data.map((c) => {
                  const dep = c.type === "deposit";
                  const tint = dep ? colors.income : colors.expense;
                  return (
                    <GlassCard key={c.id} style={styles.histCard}>
                      <View style={styles.rowBetween}>
                        <View style={{ flex: 1 }}>
                          <Text
                            style={{ color: colors.text, fontWeight: "600" }}
                          >
                            {dep ? "Setoran" : "Penarikan"}
                          </Text>
                          <Text
                            numberOfLines={1}
                            style={{
                              color: colors.textMuted,
                              fontSize: 12,
                              marginTop: 2,
                            }}
                          >
                            {formatDate(c.contribution_date)}
                            {c.note ? ` · ${c.note}` : ""}
                          </Text>
                        </View>
                        <Text style={{ color: tint, fontWeight: "800" }}>
                          {dep ? "+" : "-"}
                          {formatIDR(c.amount)}
                        </Text>
                      </View>
                    </GlassCard>
                  );
                })}
              </>
            )}
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
    paddingBottom: 60,
    maxWidth: 560,
    width: "100%",
    alignSelf: "center",
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    marginBottom: 12,
  },
  card: { padding: 18, marginBottom: 14 },
  nameRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    marginBottom: 8,
  },
  name: { fontSize: 18, fontWeight: "700", flex: 1 },
  big: { fontSize: 32, fontWeight: "800" },
  rowBetween: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 8,
  },
  actions: { flexDirection: "row", gap: 12, marginBottom: 14 },
  section: { fontSize: 16, fontWeight: "700", marginBottom: 12 },
  error: { fontSize: 13, marginBottom: 12 },
  histCard: { padding: 14, marginBottom: 8 },
});
