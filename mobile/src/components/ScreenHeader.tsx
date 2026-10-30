import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";

import { useTheme } from "@/theme/useTheme";

interface Props {
  title: string;
  subtitle?: string;
  onBack?: () => void; // tombol kembali di kiri (halaman bertumpuk)
  onClose?: () => void; // tombol tutup di kanan (modal/form)
  right?: React.ReactNode;
}

// Letakkan DI DALAM SafeAreaView, di atas ScrollView, supaya tetap terlihat saat konten di-scroll.
export function ScreenHeader({
  title,
  subtitle,
  onBack,
  onClose,
  right,
}: Props) {
  const { colors } = useTheme();
  return (
    <View style={[styles.wrap, { borderBottomColor: colors.glassBorder }]}>
      <View style={styles.inner}>
        {onBack && (
          <Pressable
            onPress={onBack}
            hitSlop={12}
            accessibilityRole="button"
            accessibilityLabel="Kembali"
            style={styles.side}
          >
            <Ionicons name="chevron-back" size={26} color={colors.text} />
          </Pressable>
        )}
        <View style={styles.center}>
          <Text
            numberOfLines={1}
            accessibilityRole="header"
            style={[styles.title, { color: colors.text }]}
          >
            {title}
          </Text>
          {!!subtitle && (
            <Text
              numberOfLines={1}
              style={{ color: colors.textMuted, fontSize: 12 }}
            >
              {subtitle}
            </Text>
          )}
        </View>
        {right}
        {onClose && (
          <Pressable
            onPress={onClose}
            hitSlop={12}
            accessibilityRole="button"
            accessibilityLabel="Tutup"
            style={styles.side}
          >
            <Ionicons name="close" size={26} color={colors.text} />
          </Pressable>
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    paddingHorizontal: 20,
    paddingVertical: 10,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  inner: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    maxWidth: 560,
    width: "100%",
    alignSelf: "center",
  },
  side: { padding: 2 },
  center: { flex: 1 },
  title: { fontSize: 20, fontWeight: "800" },
});
