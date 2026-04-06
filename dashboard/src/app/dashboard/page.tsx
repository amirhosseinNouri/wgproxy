"use client";

import { StatsCards } from "@/components/stats-cards";
import { UserTable } from "@/components/user-table";
import { AddUserDialog } from "@/components/add-user-dialog";
import { useUsers } from "@/hooks/use-users";
import { Users } from "lucide-react";

export default function DashboardPage() {
  const { data: users, isLoading, mutate } = useUsers();

  return (
    <div className="space-y-8">
      <StatsCards />

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <Users className="size-4 text-muted-foreground" />
            <h2 className="text-base font-semibold tracking-tight">Users</h2>
            {users && (
              <span className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-xs text-muted-foreground">
                {users.length}
              </span>
            )}
          </div>
          <AddUserDialog onSuccess={() => mutate()} />
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-16">
            <div className="flex items-center gap-3 text-muted-foreground">
              <div className="size-4 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-primary" />
              <span className="text-sm">Loading users...</span>
            </div>
          </div>
        ) : (
          <UserTable users={users || []} onRefresh={() => mutate()} />
        )}
      </div>
    </div>
  );
}
