"use client";

import { useState } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { EditUserDialog } from "@/components/edit-user-dialog";
import { DeleteUserDialog } from "@/components/delete-user-dialog";
import { MoreHorizontal, Pencil, Trash2, UserX, Wifi, WifiOff } from "lucide-react";
import type { User } from "@/lib/types";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(2)} ${units[i]}`;
}

interface UserTableProps {
  users: User[];
  onRefresh: () => void;
}

export function UserTable({ users, onRefresh }: UserTableProps) {
  const [editUser, setEditUser] = useState<User | null>(null);
  const [deleteUsername, setDeleteUsername] = useState<string | null>(null);

  return (
    <>
      <div className="animate-fade-in rounded-xl border border-border/50 bg-card/50 backdrop-blur-sm">
        <Table>
          <TableHeader>
            <TableRow className="border-border/50 hover:bg-transparent">
              <TableHead className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Username
              </TableHead>
              <TableHead className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Status
              </TableHead>
              <TableHead className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Upload
              </TableHead>
              <TableHead className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Download
              </TableHead>
              <TableHead className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Created
              </TableHead>
              <TableHead className="w-15" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={6}
                  className="py-16 text-center"
                >
                  <div className="flex flex-col items-center gap-2">
                    <UserX className="size-8 text-muted-foreground/40" />
                    <p className="text-sm text-muted-foreground">
                      No users found
                    </p>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              users.map((user) => (
                <TableRow
                  key={user.username}
                  className="table-row-hover border-border/30"
                >
                  <TableCell className="font-medium">{user.username}</TableCell>
                  <TableCell>
                    {!user.enabled ? (
                      <Badge variant="secondary" className="text-muted-foreground">
                        Disabled
                      </Badge>
                    ) : user.online ? (
                      <Badge
                        variant="outline"
                        className="border-teal/30 bg-teal/10 text-teal"
                      >
                        <span className="mr-1 inline-block size-1.5 rounded-full bg-teal animate-pulse" />
                        Online{user.connections > 1 ? ` (${user.connections})` : ""}
                      </Badge>
                    ) : (
                      <Badge
                        variant="outline"
                        className="border-border/50 text-muted-foreground"
                      >
                        Offline
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {formatBytes(user.upload)}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {formatBytes(user.download)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {new Date(user.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger
                        render={
                          <Button
                            variant="ghost"
                            size="icon-xs"
                            className="text-muted-foreground hover:text-foreground"
                          />
                        }
                      >
                        <MoreHorizontal className="size-3.5" />
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => setEditUser(user)}>
                          <Pencil className="mr-2 size-3.5" />
                          Edit
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          className="text-destructive"
                          onClick={() => setDeleteUsername(user.username)}
                        >
                          <Trash2 className="mr-2 size-3.5" />
                          Delete
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {editUser && (
        <EditUserDialog
          user={editUser}
          open={!!editUser}
          onOpenChange={(open) => {
            if (!open) setEditUser(null);
          }}
          onSuccess={onRefresh}
        />
      )}

      {deleteUsername && (
        <DeleteUserDialog
          username={deleteUsername}
          open={!!deleteUsername}
          onOpenChange={(open) => {
            if (!open) setDeleteUsername(null);
          }}
          onSuccess={onRefresh}
        />
      )}
    </>
  );
}
