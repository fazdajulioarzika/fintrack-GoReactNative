import { Redirect, Stack } from "expo-router";
import React from "react";
import { ActivityIndicator, View } from "react-native";

import { useAuth } from "@/context/AuthContext";

export default function AppLayout() {
  const { user, loading } = useAuth();

  if (loading) {
    return (
      <View style={{ flex: 1, alignItems: "center", justifyContent: "center" }}>
        <ActivityIndicator size="large" />
      </View>
    );
  }
  if (!user) return <Redirect href="/login" />;

  return (
    <Stack screenOptions={{ headerShown: false }}>
      <Stack.Screen name="(tabs)" />
      <Stack.Screen
        name="transaction-form"
        options={{ presentation: "modal" }}
      />
      <Stack.Screen name="budget-form" options={{ presentation: "modal" }} />
      <Stack.Screen name="goal-form" options={{ presentation: "modal" }} />
      <Stack.Screen name="goal-detail" />
    </Stack>
  );
}
