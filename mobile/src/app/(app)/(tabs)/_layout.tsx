import { Ionicons } from "@expo/vector-icons";
import { BlurView } from "expo-blur";
import { LinearGradient } from "expo-linear-gradient";
import { Tabs, useRouter } from "expo-router";
import React from "react";
import { Platform, StyleSheet, View, ColorValue } from "react-native";

import type { IconName } from "@/lib/types";
import { useTheme } from "@/theme/useTheme";

function TabBackground() {
  const { colors, isDark } = useTheme();
  if (Platform.OS === "ios") {
    return (
      <BlurView
        intensity={50}
        tint={isDark ? "dark" : "light"}
        style={StyleSheet.absoluteFill}
      />
    );
  }
  // Android/web: fallback semi-transparan (PRD 8.3)
  return (
    <View
      style={[
        StyleSheet.absoluteFill,
        {
          backgroundColor: colors.glassBgFallback,
          borderTopWidth: 1,
          borderTopColor: colors.glassBorder,
        },
      ]}
    />
  );
}

const icon = (name: IconName) =>
  function TabIcon({ color, size }: { color: ColorValue; size: number }) {
    return <Ionicons name={name} size={size} color={color} />;
  };

export default function TabsLayout() {
  const { colors } = useTheme();
  const router = useRouter();

  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarBackground: () => <TabBackground />,
        tabBarStyle: {
          position: "absolute",
          backgroundColor: "transparent",
          borderTopWidth: 0,
          elevation: 0,
        },
        tabBarLabelStyle: { fontSize: 11, fontWeight: "600" },
      }}
    >
      <Tabs.Screen
        name="index"
        options={{ title: "Beranda", tabBarIcon: icon("home") }}
      />
      <Tabs.Screen
        name="transactions"
        options={{ title: "Transaksi", tabBarIcon: icon("swap-vertical") }}
      />
      <Tabs.Screen
        name="add"
        options={{
          tabBarLabel: () => null,
          tabBarAccessibilityLabel: "Tambah transaksi",
          tabBarIcon: () => (
            <LinearGradient
              colors={[colors.primary, colors.secondary]}
              start={{ x: 0, y: 0 }}
              end={{ x: 1, y: 1 }}
              style={styles.addButton}
            >
              <Ionicons name="add" size={28} color="#FFFFFF" />
            </LinearGradient>
          ),
        }}
        listeners={{
          tabPress: (e) => {
            e.preventDefault(); // jangan pindah tab, buka form sebagai modal
            router.push("/transaction-form");
          },
        }}
      />
      <Tabs.Screen
        name="budgets"
        options={{ title: "Anggaran", tabBarIcon: icon("pie-chart") }}
      />
      <Tabs.Screen
        name="profile"
        options={{ title: "Profil", tabBarIcon: icon("person") }}
      />
    </Tabs>
  );
}

const styles = StyleSheet.create({
  addButton: {
    width: 48,
    height: 48,
    borderRadius: 24,
    alignItems: "center",
    justifyContent: "center",
    marginTop: -6,
  },
});
