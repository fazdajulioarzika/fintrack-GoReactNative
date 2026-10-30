import { Alert, Platform } from "react-native";

// Dialog konfirmasi yang jalan di HP maupun web (Alert dengan tombol tidak bekerja di web).
export function confirmAction(
  title: string,
  message: string,
  confirmText = "Hapus"
): Promise<boolean> {
  if (Platform.OS === "web") {
    return Promise.resolve(window.confirm(`${title}\n\n${message}`));
  }
  return new Promise((resolve) => {
    Alert.alert(
      title,
      message,
      [
        { text: "Batal", style: "cancel", onPress: () => resolve(false) },
        {
          text: confirmText,
          style: "destructive",
          onPress: () => resolve(true),
        },
      ],
      { cancelable: true, onDismiss: () => resolve(false) }
    );
  });
}
