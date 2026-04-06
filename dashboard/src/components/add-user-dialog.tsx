"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { createUser } from "@/lib/api";
import { generateSocksUrl } from "@/lib/socks-url";
import { toast } from "sonner";
import { Check, Copy, Loader2, Plus } from "lucide-react";

interface AddUserDialogProps {
  onSuccess: () => void;
}

export function AddUserDialog({ onSuccess }: AddUserDialogProps) {
  const [open, setOpen] = useState(false);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [configUrl, setConfigUrl] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  function handleClose(isOpen: boolean) {
    setOpen(isOpen);
    if (!isOpen) {
      setConfigUrl(null);
      setCopied(false);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    try {
      await createUser(username, password);
      toast.success(`User "${username}" created`);
      setConfigUrl(generateSocksUrl(username, password));
      setUsername("");
      setPassword("");
      onSuccess();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create user");
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
      <DialogTrigger render={<Button size="sm" className="gap-1.5" />}>
        <Plus className="size-3.5" />
        Add User
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{configUrl ? "User Created" : "Add User"}</DialogTitle>
          <DialogDescription>
            {configUrl
              ? "Copy the config URL below. It won't be shown again."
              : "Create a new proxy user account."}
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
              <Label htmlFor="new-username" className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                Username
              </Label>
              <Input
                id="new-username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="new-password" className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                Password
              </Label>
              <Input
                id="new-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? (
                <>
                  <Loader2 className="size-3.5 animate-spin" />
                  Creating...
                </>
              ) : (
                "Create User"
              )}
            </Button>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
