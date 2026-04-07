"use client";

import { Card, CardContent } from "@/components/ui/card";
import { useStats } from "@/hooks/use-stats";
import {
  ArrowUpFromLine,
  ArrowDownToLine,
  Users,
  Activity,
  Clock,
  Wifi,
} from "lucide-react";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(2)} ${units[i]}`;
}

function formatUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}d ${h}h ${m}m`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

export function StatsCards() {
  const { data, isLoading } = useStats();

  const cards = [
    {
      title: "Upload",
      value: data ? formatBytes(data.total_upload) : "-",
      icon: ArrowUpFromLine,
      accent: "text-teal",
    },
    {
      title: "Download",
      value: data ? formatBytes(data.total_download) : "-",
      icon: ArrowDownToLine,
      accent: "text-teal",
    },
    {
      title: "Total Users",
      value: data ? String(data.user_count) : "-",
      icon: Users,
      accent: "text-teal",
    },
    {
      title: "Active",
      value: data ? String(data.active_users) : "-",
      icon: Activity,
      accent: "text-teal",
    },
    {
      title: "Online",
      value: data ? String(data.online_users) : "-",
      icon: Wifi,
      accent: "text-teal",
    },
    {
      title: "Uptime",
      value: data ? formatUptime(data.uptime_seconds) : "-",
      icon: Clock,
      accent: "text-teal",
    },
  ];

  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
      {cards.map((card, i) => (
        <Card
          key={card.title}
          className={`stat-card animate-fade-in-up stagger-${i + 1} border-border/50`}
        >
          <CardContent className="flex items-start justify-between pt-1">
            <div className="space-y-1">
              <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {card.title}
              </p>
              <p className="font-mono text-2xl font-bold tracking-tight">
                {isLoading ? (
                  <span className="inline-block h-7 w-16 animate-pulse rounded bg-muted" />
                ) : (
                  card.value
                )}
              </p>
            </div>
            <div className={`rounded-lg bg-primary/10 p-2 ${card.accent}`}>
              <card.icon className="size-4" />
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
