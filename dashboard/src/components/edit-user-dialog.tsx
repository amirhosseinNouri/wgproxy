"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { updateUser } from "@/lib/api";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";
import type { User } from "@/lib/types";

interface EditUserDialogProps {
  user: User;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function EditUserDialog({
  user,
  open,
  onOpenChange,
  onSuccess,
}: EditUserDialogProps) {
  const [password, setPassword] = useState("");
  const [enabled, setEnabled] = useState(user.enabled);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    try {
      const data: { password?: string; enabled?: boolean } = {};
      if (password) data.password = password;
      if (enabled !== user.enabled) data.enabled = enabled;

      await updateUser(user.username, data);
      toast.success(`User "${user.username}" updated`);
      setPassword("");
      onOpenChange(false);
      onSuccess();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update user");
    } finally {
      setLoading(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            Edit User:{" "}
            <span className="font-mono text-primary">{user.username}</span>
          </DialogTitle>
          <DialogDescription>
            Update user settings. Leave password blank to keep current.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="edit-password" className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
              New Password
            </Label>
            <Input
              id="edit-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Leave blank to keep current"
            />
          </div>
          <div className="flex items-center justify-between rounded-lg border border-border/50 bg-muted/30 px-4 py-3">
            <Label htmlFor="edit-enabled" className="text-sm">
              Enabled
            </Label>
            <Switch
              id="edit-enabled"
              checked={enabled}
              onCheckedChange={setEnabled}
            />
          </div>
          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? (
              <>
                <Loader2 className="size-3.5 animate-spin" />
                Saving...
              </>
            ) : (
              "Save Changes"
            )}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}
