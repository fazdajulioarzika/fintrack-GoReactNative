import * as SecureStore from "expo-secure-store";
import { Platform } from "react-native";

const ACCESS_KEY = "fintrack.access";
const REFRESH_KEY = "fintrack.refresh";

// SecureStore hanya ada di HP. Di web dipakai localStorage (TIDAK aman, hanya untuk development).
const isWeb = Platform.OS === "web";

async function getItem(key: string): Promise<string | null> {
  if (isWeb)
    return typeof localStorage !== "undefined"
      ? localStorage.getItem(key)
      : null;
  return SecureStore.getItemAsync(key);
}
async function setItem(key: string, value: string): Promise<void> {
  if (isWeb) {
    localStorage.setItem(key, value);
    return;
  }
  await SecureStore.setItemAsync(key, value);
}
async function removeItem(key: string): Promise<void> {
  if (isWeb) {
    localStorage.removeItem(key);
    return;
  }
  await SecureStore.deleteItemAsync(key);
}

let accessCache: string | null = null; // access token disimpan di memori agar cepat

export const tokenStore = {
  async getAccess() {
    if (!accessCache) accessCache = await getItem(ACCESS_KEY);
    return accessCache;
  },
  getRefresh() {
    return getItem(REFRESH_KEY);
  },
  async set(access: string, refresh: string) {
    accessCache = access;
    await Promise.all([
      setItem(ACCESS_KEY, access),
      setItem(REFRESH_KEY, refresh),
    ]);
  },
  async clear() {
    accessCache = null;
    await Promise.all([removeItem(ACCESS_KEY), removeItem(REFRESH_KEY)]);
  },
};
// Preferensi non-sensitif (mis. tema). Memakai mekanisme penyimpanan yang sama.
export const prefStore = {
  get: (key: string) => getItem(`fintrack.pref.${key}`),
  set: (key: string, value: string) => setItem(`fintrack.pref.${key}`, value),
};
