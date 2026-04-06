import useSWR from "swr";
import { getUsers } from "@/lib/api";

export function useUsers() {
  return useSWR("users", getUsers, {
    refreshInterval: 30000,
    revalidateOnFocus: true,
  });
}
