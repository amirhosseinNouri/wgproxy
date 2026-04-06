import useSWR from "swr";
import { getStats } from "@/lib/api";

export function useStats() {
  return useSWR("stats", getStats, {
    refreshInterval: 30000,
    revalidateOnFocus: true,
  });
}
