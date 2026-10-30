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

const schema = z.object({
  email: z
    .string()
    .min(1, "Email wajib diisi")
    .email("Format email tidak valid"),
  password: z.string().min(1, "Password wajib diisi"),
});
type FormValues = z.infer<typeof schema>;

export default function LoginScreen() {
  const { colors } = useTheme();
  const { login } = useAuth();
  const [serverError, setServerError] = useState("");

  const {
    control,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", password: "" },
  });

  const onSubmit = async (values: FormValues) => {
    setServerError("");
    try {
      await login(values.email.trim(), values.password);
      // Redirect ke beranda ditangani otomatis oleh layout
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
              Kelola keuangan pribadimu dengan mudah
            </Text>

            <GlassCard solid style={styles.card}>
              <Text style={[styles.title, { color: colors.text }]}>Masuk</Text>

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
                    placeholder="Password Anda"
                    value={value}
                    onChangeText={onChange}
                    onBlur={onBlur}
                    error={errors.password?.message}
                    secure
                    autoCapitalize="none"
                    autoComplete="password"
                  />
                )}
              />

              {!!serverError && (
                <Text style={[styles.serverError, { color: colors.danger }]}>
                  {serverError}
                </Text>
              )}

              <Button
                title="Masuk"
                onPress={handleSubmit(onSubmit)}
                loading={isSubmitting}
              />

              <View style={styles.footer}>
                <Text style={{ color: colors.textMuted }}>
                  Belum punya akun?{" "}
                </Text>
                <Link
                  href="/register"
                  style={{ color: colors.primaryText, fontWeight: "700" }}
                >
                  Daftar
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
