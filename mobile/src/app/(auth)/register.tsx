import { zodResolver } from "@hookform/resolvers/zod";
import { Link } from "expo-router";
import React, { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import {
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { z } from "zod";

import { Button } from "@/components/ui/Button";
import { GlassCard } from "@/components/ui/GlassCard";
import { GradientBackground } from "@/components/ui/GradientBackground";
import { TextField } from "@/components/ui/TextField";
import { useAuth } from "@/context/AuthContext";
import { ApiError } from "@/lib/api";
import { useTheme } from "@/theme/useTheme";

// Aturan sama dengan backend (nama >= 2, password 8-72). Backend tetap jadi penentu akhir.
const schema = z
  .object({
    name: z
      .string()
      .trim()
      .min(2, "Nama minimal 2 karakter")
      .max(100, "Nama terlalu panjang"),
    email: z
      .string()
      .min(1, "Email wajib diisi")
      .email("Format email tidak valid"),
    password: z
      .string()
      .min(8, "Password minimal 8 karakter")
      .max(72, "Password maksimal 72 karakter"),
    confirm: z.string().min(1, "Konfirmasi password wajib diisi"),
  })
  .refine((v) => v.password === v.confirm, {
    path: ["confirm"],
    message: "Konfirmasi password tidak sama",
  });
type FormValues = z.infer<typeof schema>;

export default function RegisterScreen() {
  const { colors } = useTheme();
  const { register } = useAuth();
  const [serverError, setServerError] = useState("");

  const {
    control,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: "", email: "", password: "", confirm: "" },
  });

  const onSubmit = async (values: FormValues) => {
    setServerError("");
    try {
      await register(values.name.trim(), values.email.trim(), values.password);
    } catch (e) {
      setServerError(
        e instanceof ApiError ? e.message : "Terjadi kesalahan. Coba lagi."
      );
    }
  };

  return (
    <GradientBackground>
      <SafeAreaView style={styles.fill}>
        <KeyboardAvoidingView
          style={styles.fill}
          behavior={Platform.OS === "ios" ? "padding" : undefined}
        >
          <ScrollView
            contentContainerStyle={styles.scroll}
            keyboardShouldPersistTaps="handled"
          >
            <Text style={[styles.brand, { color: colors.text }]}>FinTrack</Text>
            <Text style={[styles.tagline, { color: colors.textMuted }]}>
              Buat akun untuk mulai mencatat
            </Text>

            <GlassCard solid style={styles.card}>
              <Text style={[styles.title, { color: colors.text }]}>Daftar</Text>

              <Controller
                control={control}
                name="name"
                render={({ field: { value, onChange, onBlur } }) => (
                  <TextField
                    label="Nama"
                    placeholder="Nama lengkap"
                    value={value}
                    onChangeText={onChange}
                    onBlur={onBlur}
                    error={errors.name?.message}
                    autoComplete="name"
                  />
                )}
              />
              <Controller
                control={control}
                name="email"
                render={({ field: { value, onChange, onBlur } }) => (
                  <TextField
                    label="Email"
                    placeholder="nama@email.com"
                    value={value}
                    onChangeText={onChange}
                    onBlur={onBlur}
                    error={errors.email?.message}
                    autoCapitalize="none"
                    autoCorrect={false}
                    keyboardType="email-address"
                    autoComplete="email"
                  />
                )}
              />
              <Controller
                control={control}
                name="password"
                render={({ field: { value, onChange, onBlur } }) => (
                  <TextField
                    label="Password"
                    placeholder="Minimal 8 karakter"
                    value={value}
                    onChangeText={onChange}
                    onBlur={onBlur}
                    error={errors.password?.message}
                    secure
                    autoCapitalize="none"
                  />
                )}
              />
              <Controller
                control={control}
                name="confirm"
                render={({ field: { value, onChange, onBlur } }) => (
                  <TextField
                    label="Ulangi password"
                    placeholder="Ketik ulang password"
                    value={value}
                    onChangeText={onChange}
                    onBlur={onBlur}
                    error={errors.confirm?.message}
                    secure
                    autoCapitalize="none"
                  />
                )}
              />

              {!!serverError && (
                <Text style={[styles.serverError, { color: colors.danger }]}>
                  {serverError}
                </Text>
              )}

              <Button
                title="Buat akun"
                onPress={handleSubmit(onSubmit)}
                loading={isSubmitting}
              />

              <View style={styles.footer}>
                <Text style={{ color: colors.textMuted }}>
                  Sudah punya akun?{" "}
                </Text>
                <Link
                  href="/login"
                  style={{ color: colors.primary, fontWeight: "700" }}
                >
                  Masuk
                </Link>
              </View>
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
    flexGrow: 1,
    justifyContent: "center",
    padding: 24,
    maxWidth: 480,
    width: "100%",
    alignSelf: "center",
  },
  brand: { fontSize: 34, fontWeight: "800", textAlign: "center" },
  tagline: {
    fontSize: 14,
    textAlign: "center",
    marginTop: 4,
    marginBottom: 28,
  },
  card: { padding: 22 },
  title: { fontSize: 22, fontWeight: "700", marginBottom: 18 },
  serverError: { fontSize: 13, marginBottom: 12 },
  footer: { flexDirection: "row", justifyContent: "center", marginTop: 18 },
});
