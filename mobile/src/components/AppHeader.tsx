import { Ionicons } from "@expo/vector-icons";
import { BlurView } from "expo-blur";
import { LinearGradient } from "expo-linear-gradient";
import { useRouter } from "expo-router";
import React from "react";
import { Platform, Pressable, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { useAuth } from "@/context/AuthContext";
import type { IconName } from "@/lib/types";
import { useTheme } from "@/theme/useTheme";

export function HeaderIconButton({
  icon,
  label,
  onPress,
}: {
  icon: IconName;
  label: string;
  onPress: () => void;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      onPress={onPress}
      hitSlop={6}
      accessibilityRole="button"
      accessibilityLabel={label}
      style={[
        styles.iconBtn,
        {
          backgroundColor: colors.glassBgFallback,
          borderColor: colors.glassBorder,
        },
      ]}
    >
      <Ionicons name={icon} size={20} color={colors.text} />
    </Pressable>
  );
}

interface Props {
  title: string;
  right?: React.ReactNode;
  showAvatar?: boolean;
}

// Header untuk layar tab. Sudah memperhitungkan area aman atas (notch/status bar).
export function AppHeader({ title, right, showAvatar = true }: Props) {
  const { colors, isDark } = useTheme();
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const { user } = useAuth();
  const initial = (user?.name?.trim()?.[0] ?? "?").toUpperCase();

  const content = (
    <View style={styles.inner}>
      <View style={styles.left}>
        <LinearGradient
          colors={[colors.primary, colors.secondary]}
          start={{ x: 0, y: 0 }}
          end={{ x: 1, y: 1 }}
          style={styles.logo}
        >
          <Ionicons name="wallet" size={18} color="#FFFFFF" />
        </LinearGradient>
        <View>
          <Text style={[styles.brand, { color: colors.textMuted }]}>
            FinTrack
          </Text>
          <Text
            accessibilityRole="header"
            style={[styles.title, { color: colors.text }]}
          >
            {title}
          </Text>
        </View>
      </View>
      <View style={styles.right}>
        {right}
        {showAvatar && (
          <Pressable
            onPress={() => router.push("/profile")}
            accessibilityRole="button"
            accessibilityLabel="Buka profil"
          >
            <LinearGradient
              colors={[colors.primary, colors.secondary]}
              start={{ x: 0, y: 0 }}
              end={{ x: 1, y: 1 }}
              style={styles.avatar}
            >
              <Text style={styles.avatarText}>{initial}</Text>
            </LinearGradient>
          </Pressable>
        )}
      </View>
    </View>
  );

  const wrapStyle = [
    styles.wrap,
    { paddingTop: insets.top + 8, borderBottomColor: colors.glassBorder },
  ];

  // Blur asli hanya di iOS; Android dan web memakai fallback semi-transparan (PRD 8.3)
  if (Platform.OS === "ios") {
    return (
      <BlurView
        intensity={50}
        tint={isDark ? "dark" : "light"}
        style={[wrapStyle, { backgroundColor: colors.glassBg }]}
      >
        {content}
      </BlurView>
    );
  }
  return (
    <View style={[wrapStyle, { backgroundColor: colors.glassBgFallback }]}>
      {content}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    paddingBottom: 10,
    paddingHorizontal: 20,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  inner: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    maxWidth: 560,
    width: "100%",
    alignSelf: "center",
  },
  left: { flexDirection: "row", alignItems: "center", gap: 12, flex: 1 },
  right: { flexDirection: "row", alignItems: "center", gap: 10 },
  logo: {
    width: 38,
    height: 38,
    borderRadius: 12,
    alignItems: "center",
    justifyContent: "center",
  },
  brand: { fontSize: 11, fontWeight: "600", letterSpacing: 0.5 },
  title: { fontSize: 20, fontWeight: "800" },
  avatar: {
    width: 38,
    height: 38,
    borderRadius: 19,
    alignItems: "center",
    justifyContent: "center",
  },
  avatarText: { color: "#FFFFFF", fontSize: 16, fontWeight: "800" },
  iconBtn: {
    width: 38,
    height: 38,
    borderRadius: 19,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
  },
});
