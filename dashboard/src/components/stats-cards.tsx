"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useStats } from "@/hooks/use-stats";

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
      title: "Total Upload",
      value: data ? formatBytes(data.total_upload) : "-",
    },
    {
      title: "Total Download",
      value: data ? formatBytes(data.total_download) : "-",
    },
    {
      title: "Total Users",
      value: data ? String(data.user_count) : "-",
    },
    {
      title: "Active Users",
      value: data ? String(data.active_users) : "-",
    },
    {
      title: "Uptime",
      value: data ? formatUptime(data.uptime_seconds) : "-",
    },
  ];

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
      {cards.map((card) => (
        <Card key={card.title}>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {card.title}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {isLoading ? "..." : card.value}
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
