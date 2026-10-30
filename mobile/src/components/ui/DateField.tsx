import { Ionicons } from "@expo/vector-icons";
import DateTimePicker, {
  DateTimePickerAndroid,
} from "@react-native-community/datetimepicker";
import React from "react";
import { Platform, Pressable, StyleSheet, Text, View } from "react-native";

import { TextField } from "@/components/ui/TextField";
import { dateToStr, formatDay, strToDate, todayStr } from "@/lib/date";
import { useTheme } from "@/theme/useTheme";

interface Props {
  label: string;
  value: string; // 'YYYY-MM-DD', atau '' jika kosong (hanya untuk clearable)
  onChange: (v: string) => void;
  clearable?: boolean;
  minimumDate?: Date;
  maximumDate?: Date;
  error?: string;
}

export function DateField({
  label,
  value,
  onChange,
  clearable = false,
  minimumDate,
  maximumDate,
  error,
}: Props) {
  const { colors, isDark } = useTheme();

  // Pemilih tanggal native tidak tersedia di web: pakai input teks
  if (Platform.OS === "web") {
    return (
      <TextField
        label={`${label} (TTTT-BB-HH)`}
        value={value}
        onChangeText={onChange}
        placeholder="2026-10-29"
        maxLength={10}
        error={error}
      />
    );
  }

  const hasValue = value !== "";

  const openAndroid = () => {
    DateTimePickerAndroid.open({
      value: strToDate(value),
      mode: "date",
      minimumDate,
      maximumDate,
      onValueChange: (_e, d) => onChange(dateToStr(d)),
    });
  };

  return (
    <View style={styles.wrap}>
      <Text style={[styles.label, { color: colors.text }]}>{label}</Text>
      <View
        style={[
          styles.box,
          {
            backgroundColor: colors.inputBg,
            borderColor: error ? colors.danger : colors.inputBorder,
          },
        ]}
      >
        <Ionicons name="calendar-outline" size={20} color={colors.textMuted} />

        {Platform.OS === "ios" && hasValue ? (
          <View style={styles.iosPicker}>
            <DateTimePicker
              value={strToDate(value)}
              mode="date"
              display="compact"
              themeVariant={isDark ? "dark" : "light"}
              minimumDate={minimumDate}
              maximumDate={maximumDate}
              onValueChange={(_e, d) => onChange(dateToStr(d))}
            />
          </View>
        ) : (
          <Pressable
            style={styles.textBtn}
            onPress={
              Platform.OS === "ios" ? () => onChange(todayStr()) : openAndroid
            }
            accessibilityRole="button"
            accessibilityLabel={label}
          >
            <Text
              style={{
                color: hasValue ? colors.text : colors.textMuted,
                fontSize: 16,
              }}
            >
              {hasValue ? formatDay(value) : "Pilih tanggal"}
            </Text>
          </Pressable>
        )}

        {clearable && hasValue && (
          <Pressable
            onPress={() => onChange("")}
            hitSlop={10}
            accessibilityLabel={`Hapus ${label}`}
          >
            <Ionicons name="close-circle" size={20} color={colors.textMuted} />
          </Pressable>
        )}
      </View>
      {!!error && (
        <Text style={[styles.error, { color: colors.danger }]}>{error}</Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { marginBottom: 14 },
  label: { fontSize: 14, fontWeight: "600", marginBottom: 6 },
  box: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    borderWidth: 1,
    borderRadius: 14,
    paddingHorizontal: 14,
    height: 50,
  },
  textBtn: { flex: 1, height: "100%", justifyContent: "center" },
  iosPicker: { flex: 1, alignItems: "flex-start" },
  error: { fontSize: 12, marginTop: 4 },
});
