import type { Budget } from "@/lib/types";
import type { Colors } from "@/theme/colors";

export function budgetColor(status: Budget["status"], colors: Colors): string {
  if (status === "exceeded") return colors.expense;
  if (status === "warning") return colors.warning;
  return colors.primaryText;
}

export function budgetLabel(status: Budget["status"]): string {
  if (status === "exceeded") return "Terlampaui";
  if (status === "warning") return "Hampir habis";
  return "";
}
