import type { Colors } from "@/theme/colors";
import type { Budget } from "@/lib/types";

export function budgetColor(status: Budget["status"], colors: Colors): string {
  if (status === "exceeded") return colors.expense;
  if (status === "warning") return "#F59E0B";
  return colors.primary;
}

export function budgetLabel(status: Budget["status"]): string {
  if (status === "exceeded") return "Terlampaui";
  if (status === "warning") return "Hampir habis";
  return "";
}
