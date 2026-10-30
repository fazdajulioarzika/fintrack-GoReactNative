import type { QueryClient } from "@tanstack/react-query";

// Panggil setelah transaksi berubah: semua angka turunan harus ikut diperbarui.
export function invalidateFinance(qc: QueryClient) {
  for (const key of ["dashboard", "transactions", "budgets", "analytics"]) {
    qc.invalidateQueries({ queryKey: [key] });
  }
}
