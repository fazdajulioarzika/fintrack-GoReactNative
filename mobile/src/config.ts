// Variabel EXPO_PUBLIC_* dibaca saat build. Ubah .env lalu restart dengan: npx expo start -c
export const API_URL =
  process.env.EXPO_PUBLIC_API_URL ?? "http://192.168.0.107:8080/api/v1";
