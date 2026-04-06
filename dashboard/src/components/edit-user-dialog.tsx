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
import { generateSocksUrl } from "@/lib/socks-url";
import { toast } from "sonner";
import { Check, Copy, Loader2 } from "lucide-react";
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
  const [configUrl, setConfigUrl] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  function handleClose(isOpen: boolean) {
    onOpenChange(isOpen);
    if (!isOpen) {
      setConfigUrl(null);
      setCopied(false);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    try {
      const data: { password?: string; enabled?: boolean } = {};
      if (password) data.password = password;
      if (enabled !== user.enabled) data.enabled = enabled;

      await updateUser(user.username, data);
      toast.success(`User "${user.username}" updated`);

      if (password) {
        setConfigUrl(generateSocksUrl(user.username, password));
      } else {
        setPassword("");
        handleClose(false);
      }
      onSuccess();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update user");
    } finally {
      setLoading(false);
    }
  }

  function handleCopy() {
    if (!configUrl) return;
    navigator.clipboard.writeText(configUrl);
    setCopied(true);
    toast.success("Config URL copied to clipboard");
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {configUrl ? "Password Updated" : (
              <>
                Edit User:{" "}
                <span className="font-mono text-primary">{user.username}</span>
              </>
            )}
          </DialogTitle>
          <DialogDescription>
            {configUrl
              ? "Copy the new config URL below. It won't be shown again."
              : "Update user settings. Leave password blank to keep current."}
          </DialogDescription>
        </DialogHeader>
        {configUrl ? (
          <div className="space-y-3">
            <div className="flex items-center gap-2 rounded-lg border border-border/50 bg-muted/30 p-3">
              <code className="flex-1 break-all text-xs text-muted-foreground">
                {configUrl}
              </code>
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={handleCopy}
                className="shrink-0"
              >
                {copied ? (
                  <Check className="size-3.5 text-teal" />
                ) : (
                  <Copy className="size-3.5" />
                )}
              </Button>
            </div>
            <Button
              className="w-full"
              onClick={() => handleClose(false)}
            >
              Done
            </Button>
          </div>
        ) : (
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
        )}
      </DialogContent>
    </Dialog>
  );
}
