import type { Ionicons } from "@expo/vector-icons";
import type React from "react";

export type IconName = React.ComponentProps<typeof Ionicons>["name"];
export type TxType = "income" | "expense";

export interface Category {
  id: string;
  name: string;
  type: TxType;
  icon: string;
}

export interface Transaction {
  id: string;
  type: TxType;
  amount: number;
  category: { id: string; name: string; icon: string } | null;
  note: string;
  payment_method: string;
  transaction_date: string;
}

export interface TxList {
  data: Transaction[];
  meta: { page: number; limit: number; total: number; total_pages: number };
}

export interface Summary {
  balance: number;
  month_income: number;
  month_expense: number;
  month_net: number;
  transaction_count: number;
}

export interface CashflowPoint {
  date: string;
  income: number;
  expense: number;
  net: number;
}

export interface CategoryShare {
  category_id: string;
  name: string;
  icon: string;
  total: number;
  percent: number;
}

export interface Budget {
  id: string;
  category: { id: string; name: string; icon: string } | null;
  limit_amount: number;
  spent_amount: number;
  usage_percent: number;
  status: "normal" | "warning" | "exceeded";
}
export interface BudgetList {
  total_limit: number;
  total_spent: number;
  data: Budget[];
}

export interface SavingsGoal {
  id: string;
  name: string;
  target_amount: number;
  saved_amount: number;
  remaining_amount: number;
  progress_percent: number;
  target_date: string | null;
  icon: string;
  status: "active" | "achieved";
}

export interface GoalList {
  total_target: number;
  total_saved: number;
  data: SavingsGoal[];
}

export interface Contribution {
  id: string;
  goal_id: string;
  type: "deposit" | "withdraw";
  amount: number;
  contribution_date: string;
  note: string;
}
export interface RangeInfo {
  label: string;
  start_date: string;
  end_date: string;
}

export interface AnalyticsSummary {
  period: RangeInfo;
  total_income: number;
  total_expense: number;
  net: number;
  transaction_count: number;
  expense_by_category: CategoryShare[];
  expense_by_category_total: number;
}

export interface TrendResult {
  period: RangeInfo;
  granularity: "day" | "month";
  data: CashflowPoint[];
}

export interface ChangeInfo {
  amount: number;
  percent: number | null; // null jika periode pembanding bernilai 0
}

export interface CompareBlock {
  period: RangeInfo;
  income: number;
  expense: number;
  net: number;
  transaction_count: number;
}

export interface CompareResult {
  current: CompareBlock;
  previous: CompareBlock;
  change: { income: ChangeInfo; expense: ChangeInfo; net: ChangeInfo };
}
