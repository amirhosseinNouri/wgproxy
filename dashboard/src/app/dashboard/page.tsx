"use client";

import { StatsCards } from "@/components/stats-cards";
import { UserTable } from "@/components/user-table";
import { AddUserDialog } from "@/components/add-user-dialog";
import { useUsers } from "@/hooks/use-users";

export default function DashboardPage() {
  const { data: users, isLoading, mutate } = useUsers();

  return (
    <div className="space-y-6">
      <StatsCards />

      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">Users</h2>
        <AddUserDialog onSuccess={() => mutate()} />
      </div>

      {isLoading ? (
        <p className="text-muted-foreground">Loading users...</p>
      ) : (
        <UserTable users={users || []} onRefresh={() => mutate()} />
      )}
    </div>
  );
}
