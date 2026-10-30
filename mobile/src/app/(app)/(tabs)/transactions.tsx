import { useInfiniteQuery } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import React, { useEffect, useState } from "react";
import {
  ActivityIndicator,
  FlatList,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";
import { AppHeader } from "@/components/AppHeader";
import { TransactionRow } from "@/components/TransactionRow";
import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { GlassCard } from "@/components/ui/GlassCard";
import { GradientBackground } from "@/components/ui/GradientBackground";
import { api, ApiError } from "@/lib/api";
import type { TxList } from "@/lib/types";
import { useTheme } from "@/theme/useTheme";

type Filter = "all" | "income" | "expense";

export default function TransactionsScreen() {
  const { colors } = useTheme();
  const router = useRouter();
  const [filter, setFilter] = useState<Filter>("all");
  const [search, setSearch] = useState("");
  const [q, setQ] = useState("");

  // Tunggu 400 ms setelah berhenti mengetik agar tidak memanggil server di setiap huruf
  useEffect(() => {
    const t = setTimeout(() => setQ(search.trim()), 400);
    return () => clearTimeout(t);
  }, [search]);

  const query = useInfiniteQuery({
    queryKey: ["transactions", "list", filter, q],
    initialPageParam: 1,
    queryFn: ({ pageParam }) => {
      let url = `/transactions?page=${pageParam}&limit=20`;
      if (filter !== "all") url += `&type=${filter}`;
      if (q) url += `&q=${encodeURIComponent(q)}`;
      return api.get<TxList>(url);
    },
    getNextPageParam: (last) =>
      last.meta.page < last.meta.total_pages ? last.meta.page + 1 : undefined,
  });

  const items = query.data?.pages.flatMap((p) => p.data) ?? [];

  return (
    <GradientBackground>
      <View style={styles.fill}>
        <AppHeader title="Transaksi" />
        <View style={styles.header}>
          <TextInput
            value={search}
            onChangeText={setSearch}
            placeholder="Cari catatan..."
            placeholderTextColor={colors.textMuted}
            style={[
              styles.search,
              {
                backgroundColor: colors.inputBg,
                borderColor: colors.inputBorder,
                color: colors.text,
              },
            ]}
          />
          <View style={styles.chips}>
            <Chip
              label="Semua"
              selected={filter === "all"}
              onPress={() => setFilter("all")}
            />
            <Chip
              label="Pemasukan"
              selected={filter === "income"}
              onPress={() => setFilter("income")}
              tint={colors.income}
            />
            <Chip
              label="Pengeluaran"
              selected={filter === "expense"}
              onPress={() => setFilter("expense")}
              tint={colors.expense}
            />
          </View>
        </View>

        {query.isLoading ? (
          <ActivityIndicator style={{ marginTop: 40 }} />
        ) : query.error ? (
          <View style={styles.pad}>
            <Text style={{ color: colors.danger, marginBottom: 10 }}>
              {query.error instanceof ApiError
                ? query.error.message
                : "Gagal memuat transaksi"}
            </Text>
            <Button
              title="Coba lagi"
              variant="outline"
              onPress={() => query.refetch()}
            />
          </View>
        ) : (
          <FlatList
            data={items}
            keyExtractor={(t) => t.id}
            contentContainerStyle={styles.list}
            refreshing={query.isRefetching && !query.isFetchingNextPage}
            onRefresh={() => query.refetch()}
            onEndReached={() =>
              query.hasNextPage &&
              !query.isFetchingNextPage &&
              query.fetchNextPage()
            }
            onEndReachedThreshold={0.4}
            renderItem={({ item }) => (
              <GlassCard style={styles.item}>
                <TransactionRow
                  tx={item}
                  onPress={() =>
                    router.push({
                      pathname: "/transaction-form",
                      params: { id: item.id },
                    })
                  }
                />
              </GlassCard>
            )}
            ListEmptyComponent={
              <Text
                style={{
                  color: colors.textMuted,
                  textAlign: "center",
                  marginTop: 40,
                }}
              >
                {q || filter !== "all"
                  ? "Tidak ada transaksi yang cocok."
                  : "Belum ada transaksi. Tekan tombol + untuk menambah."}
              </Text>
            }
            ListFooterComponent={
              query.isFetchingNextPage ? (
                <ActivityIndicator style={{ margin: 16 }} />
              ) : null
            }
          />
        )}
      </View>
    </GradientBackground>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  header: {
    paddingHorizontal: 20,
    paddingTop: 8,
    maxWidth: 560,
    width: "100%",
    alignSelf: "center",
  },
  title: { fontSize: 26, fontWeight: "800", marginBottom: 12 },
  search: {
    height: 46,
    borderRadius: 14,
    borderWidth: 1,
    paddingHorizontal: 14,
    fontSize: 15,
    marginBottom: 12,
  },
  chips: { flexDirection: "row", gap: 8, marginBottom: 8 },
  list: {
    paddingHorizontal: 20,
    paddingTop: 8,
    paddingBottom: 120,
    maxWidth: 560,
    width: "100%",
    alignSelf: "center",
  },
  item: { paddingHorizontal: 14, marginBottom: 10 },
  pad: { padding: 20 },
});
