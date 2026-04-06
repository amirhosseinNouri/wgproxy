import useSWR from "swr";
import { getStats } from "@/lib/api";

export function useStats() {
  return useSWR("stats", getStats, {
    refreshInterval: 15000,
    revalidateOnFocus: true,
  });
}
