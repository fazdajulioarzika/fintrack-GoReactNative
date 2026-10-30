import { Ionicons } from "@expo/vector-icons";
import React, { useState } from "react";
import {
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  TextInputProps,
  View,
} from "react-native";

import { useTheme } from "@/theme/useTheme";

interface Props extends TextInputProps {
  label: string;
  error?: string;
  secure?: boolean; // field password dengan tombol tampilkan/sembunyikan
}

export function TextField({
  label,
  error,
  secure = false,
  style,
  ...rest
}: Props) {
  const { colors } = useTheme();
  const [hidden, setHidden] = useState(true);

  return (
    <View style={styles.wrap}>
      <Text style={[styles.label, { color: colors.text }]}>{label}</Text>
      <View
        style={[
          styles.inputRow,
          {
            backgroundColor: colors.inputBg,
            borderColor: error ? colors.danger : colors.inputBorder,
          },
        ]}
      >
        <TextInput
          {...rest}
          secureTextEntry={secure && hidden}
          placeholderTextColor={colors.textMuted}
          style={[styles.input, { color: colors.text }, style]}
        />
        {secure && (
          <Pressable
            onPress={() => setHidden((h) => !h)}
            hitSlop={10}
            accessibilityLabel={
              hidden ? "Tampilkan password" : "Sembunyikan password"
            }
          >
            <Ionicons
              name={hidden ? "eye-outline" : "eye-off-outline"}
              size={20}
              color={colors.textMuted}
            />
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
  inputRow: {
    flexDirection: "row",
    alignItems: "center",
    borderWidth: 1,
    borderRadius: 14,
    paddingHorizontal: 14,
    height: 50,
  },
  input: { flex: 1, fontSize: 16 },
  error: { fontSize: 12, marginTop: 4 },
});
