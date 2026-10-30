import { Ionicons } from "@expo/vector-icons";
import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import React, { useState } from "react";
import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import { AppHeader } from "@/components/AppHeader";
import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { GlassCard } from "@/components/ui/GlassCard";
import { GradientBackground } from "@/components/ui/GradientBackground";
import { TextField } from "@/components/ui/TextField";
import { useAuth, type User } from "@/context/AuthContext";
import { api, ApiError } from "@/lib/api";
import { confirmAction } from "@/lib/confirm";
import { useTheme, useThemePreference } from "@/theme/useTheme";

const ZONES = [
  { label: "WIB", value: "Asia/Jakarta" },
  { label: "WITA", value: "Asia/Makassar" },
  { label: "WIT", value: "Asia/Jayapura" },
];

export default function ProfileScreen() {
  const { colors } = useTheme();
  const { pref, setPref } = useThemePreference();
  const { user, logout, updateUser, deleteAccount } = useAuth();
  const router = useRouter();
  const qc = useQueryClient();

  const [name, setName] = useState(user?.name ?? "");
  const [nameMsg, setNameMsg] = useState<{ ok: boolean; text: string } | null>(
    null
  );
  const [savingName, setSavingName] = useState(false);
  const [zoneError, setZoneError] = useState("");
  const [zoneSaving, setZoneSaving] = useState(false);

  const [showDelete, setShowDelete] = useState(false);
  const [password, setPassword] = useState("");
  const [deleteError, setDeleteError] = useState("");
  const [deleting, setDeleting] = useState(false);

  const nameChanged = name.trim() !== (user?.name ?? "");

  const saveName = async () => {
    setNameMsg(null);
    const trimmed = name.trim();
    if (trimmed.length < 2)
      return setNameMsg({ ok: false, text: "Nama minimal 2 karakter" });
    setSavingName(true);
    try {
      const res = await api.patch<{ user: User }>("/auth/me", {
        name: trimmed,
      });
      updateUser(res.user);
      setName(res.user.name);
      setNameMsg({ ok: true, text: "Nama berhasil diperbarui" });
    } catch (e) {
      setNameMsg({
        ok: false,
        text: e instanceof ApiError ? e.message : "Gagal menyimpan. Coba lagi.",
      });
    } finally {
      setSavingName(false);
    }
  };

  const changeZone = async (tz: string) => {
    if (tz === user?.timezone || zoneSaving) return;
    setZoneError("");
    setZoneSaving(true);
    try {
      const res = await api.patch<{ user: User }>("/auth/me", { timezone: tz });
      updateUser(res.user);
      qc.invalidateQueries(); // batas bulan berubah, semua data harus dihitung ulang
    } catch (e) {
      setZoneError(
        e instanceof ApiError ? e.message : "Gagal mengubah zona waktu."
      );
    } finally {
      setZoneSaving(false);
    }
  };

  const doDelete = async () => {
    setDeleteError("");
    if (!password) return setDeleteError("Masukkan password untuk konfirmasi");
    const ok = await confirmAction(
      "Hapus akun permanen?",
      "Seluruh transaksi, anggaran, dan target tabungan Anda akan dihapus dan tidak dapat dikembalikan.",
      "Hapus akun"
    );
    if (!ok) return;
    setDeleting(true);
    try {
      await deleteAccount(password); // setelah berhasil, layout otomatis kembali ke halaman login
    } catch (e) {
      setDeleteError(
        e instanceof ApiError ? e.message : "Gagal menghapus akun. Coba lagi."
      );
      setDeleting(false);
    }
  };

  return (
    <GradientBackground>
      <View style={styles.fill}>
        <AppHeader title="Profil" />
        <ScrollView
          contentContainerStyle={styles.scroll}
          keyboardShouldPersistTaps="handled"
        >
          <GlassCard solid style={styles.card}>
            <Text style={[styles.section, { color: colors.text }]}>Akun</Text>
            <TextField
              label="Nama"
              value={name}
              onChangeText={setName}
              maxLength={100}
              autoComplete="name"
            />
            <Text style={{ color: colors.textMuted, fontSize: 12 }}>Email</Text>
            <Text
              style={{
                color: colors.text,
                fontSize: 16,
                fontWeight: "600",
                marginTop: 2,
                marginBottom: 14,
              }}
            >
              {user?.email}
            </Text>
            {!!nameMsg && (
              <Text
                style={{
                  color: nameMsg.ok ? colors.income : colors.danger,
                  fontSize: 13,
                  marginBottom: 10,
                }}
              >
                {nameMsg.text}
              </Text>
            )}
            <Button
              title="Simpan nama"
              onPress={saveName}
              loading={savingName}
              disabled={!nameChanged}
            />
          </GlassCard>

          <GlassCard style={styles.card}>
            <Text style={[styles.section, { color: colors.text }]}>
              Tampilan
            </Text>
            <View style={styles.chips}>
              <Chip
                label="Sistem"
                selected={pref === "system"}
                onPress={() => setPref("system")}
              />
              <Chip
                label="Terang"
                selected={pref === "light"}
                onPress={() => setPref("light")}
              />
              <Chip
                label="Gelap"
                selected={pref === "dark"}
                onPress={() => setPref("dark")}
              />
            </View>
          </GlassCard>

          <GlassCard style={styles.card}>
            <Text style={[styles.section, { color: colors.text }]}>
              Zona waktu
            </Text>
            <Text
              style={{ color: colors.textMuted, fontSize: 12, marginTop: 4 }}
            >
              Menentukan batas hari dan bulan pada ringkasan, anggaran, dan
              analisis.
            </Text>
            <View style={styles.chips}>
              {ZONES.map((z) => (
                <Chip
                  key={z.value}
                  label={z.label}
                  selected={user?.timezone === z.value}
                  onPress={() => changeZone(z.value)}
                />
              ))}
            </View>
            {!!user?.timezone &&
              !ZONES.some((z) => z.value === user.timezone) && (
                <Text
                  style={{
                    color: colors.textMuted,
                    fontSize: 12,
                    marginTop: 8,
                  }}
                >
                  Saat ini: {user.timezone}
                </Text>
              )}
            {!!zoneError && (
              <Text
                style={{ color: colors.danger, fontSize: 13, marginTop: 8 }}
              >
                {zoneError}
              </Text>
            )}
          </GlassCard>

          <Pressable
            onPress={() => router.push("/analytics")}
            accessibilityRole="button"
          >
            <GlassCard style={[styles.card, styles.linkRow]}>
              <Ionicons name="stats-chart" size={20} color={colors.primary} />
              <Text style={{ color: colors.text, fontWeight: "600", flex: 1 }}>
                Analisis keuangan
              </Text>
              <Ionicons
                name="chevron-forward"
                size={20}
                color={colors.textMuted}
              />
            </GlassCard>
          </Pressable>

          <Button title="Keluar" variant="outline" onPress={logout} />

          <GlassCard solid style={[styles.card, { marginTop: 24 }]}>
            <Text style={[styles.section, { color: colors.danger }]}>
              Zona berbahaya
            </Text>
            {!showDelete ? (
              <>
                <Text
                  style={{
                    color: colors.textMuted,
                    fontSize: 13,
                    marginVertical: 10,
                  }}
                >
                  Menghapus akun akan menghapus semua data keuangan Anda secara
                  permanen.
                </Text>
                <Button
                  title="Hapus akun"
                  variant="outline"
                  onPress={() => setShowDelete(true)}
                />
              </>
            ) : (
              <View style={{ marginTop: 10 }}>
                <TextField
                  label="Masukkan password untuk konfirmasi"
                  value={password}
                  onChangeText={setPassword}
                  secure
                  autoCapitalize="none"
                  placeholder="Password Anda"
                />
                {!!deleteError && (
                  <Text
                    style={{
                      color: colors.danger,
                      fontSize: 13,
                      marginBottom: 10,
                    }}
                  >
                    {deleteError}
                  </Text>
                )}
                <Button
                  title="Hapus akun permanen"
                  onPress={doDelete}
                  loading={deleting}
                />
                <View style={{ marginTop: 10 }}>
                  <Button
                    title="Batal"
                    variant="outline"
                    disabled={deleting}
                    onPress={() => {
                      setShowDelete(false);
                      setPassword("");
                      setDeleteError("");
                    }}
                  />
                </View>
              </View>
            )}
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
  title: { fontSize: 26, fontWeight: "800", marginBottom: 16 },
  card: { padding: 18, marginBottom: 14 },
  section: { fontSize: 16, fontWeight: "700", marginBottom: 10 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8, marginTop: 8 },
  linkRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 12,
    paddingVertical: 14,
  },
});
