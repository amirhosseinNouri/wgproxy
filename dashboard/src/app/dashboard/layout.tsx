"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { isAuthenticated, logout } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { LogOut, Shield } from "lucide-react";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
    }
  }, [router]);

  return (
    <div className="flex min-h-screen flex-col grid-bg">
      <header className="sticky top-0 z-40 border-b border-border/50 bg-background/80 backdrop-blur-xl">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-6">
          <div className="flex items-center gap-2.5">
            <div className="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <Shield className="size-4" />
            </div>
            <div className="flex flex-col">
              <span className="text-sm font-semibold tracking-tight">
                WG Proxy
              </span>
              <span className="text-[10px] font-medium uppercase tracking-widest text-muted-foreground">
                Dashboard
              </span>
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={logout}
            className="gap-1.5 text-muted-foreground hover:text-foreground"
          >
            <LogOut className="size-3.5" />
            Logout
          </Button>
        </div>
      </header>
      <main className="mx-auto w-full max-w-7xl flex-1 px-6 py-8">
        {children}
      </main>
    </div>
  );
}
