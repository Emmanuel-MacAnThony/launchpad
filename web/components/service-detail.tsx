"use client";

import { useEffect, useState } from "react";
import { Globe, GitBranch, Server, Link, Activity, User } from "lucide-react";
import { api, type Service, type Deploy } from "@/lib/api";
import { DeployList } from "@/components/deploy-list";
import { SlotBadge } from "@/components/status-badge";
import { relativeTime, cn } from "@/lib/utils";

function InfoRow({
  icon: Icon,
  label,
  value,
  dim = false,
}: {
  icon: React.ElementType;
  label: string;
  value: string;
  dim?: boolean;
}) {
  return (
    <div className="flex items-start gap-3 py-2.5 border-b border-[#1c1c1f] last:border-b-0">
      <Icon className="w-3.5 h-3.5 text-[#52525b] mt-0.5 shrink-0" />
      <span className="text-[12px] text-[#52525b] w-20 shrink-0">{label}</span>
      <span
        className={cn(
          "text-[12px] font-mono break-all leading-relaxed",
          dim ? "text-[#52525b]" : "text-[#a1a1aa]"
        )}
      >
        {value}
      </span>
    </div>
  );
}

function ActiveBanner({ deploys }: { deploys: Deploy[] }) {
  const active = deploys.find((d) => d.Status === "active");

  if (!active) {
    return (
      <div className="flex items-center gap-3 px-4 py-3 rounded-lg border border-dashed border-[#27272a] text-[12px] text-[#52525b]">
        <span className="w-2 h-2 rounded-full bg-[#27272a]" />
        No active deployment
      </div>
    );
  }

  const isBlue  = active.Slot === "blue";
  const color   = isBlue ? "#60a5fa" : "#2dd4bf";
  const bgColor = isBlue ? "rgba(96,165,250,0.05)" : "rgba(45,212,191,0.05)";
  const bdrColor = isBlue ? "rgba(96,165,250,0.2)" : "rgba(45,212,191,0.2)";

  return (
    <div
      className="flex items-center justify-between px-4 py-3 rounded-lg border"
      style={{ borderColor: bdrColor, backgroundColor: bgColor }}
    >
      <div className="flex items-center gap-3">
        <span
          className="w-2 h-2 rounded-full shrink-0 pulse-active"
          style={{ backgroundColor: color }}
        />
        <div>
          <div className="flex items-center gap-2">
            <SlotBadge slot={active.Slot} />
            <code className="text-[12px]" style={{ color }}>
              {active.CommitSHA.slice(0, 7)}
            </code>
            <span className="text-[12px] text-[#52525b] truncate max-w-[240px]">
              {active.CommitMessage}
            </span>
          </div>
        </div>
      </div>
      <span className="text-[12px] text-[#52525b] whitespace-nowrap ml-4">
        {relativeTime(active.FinishedAt ?? active.CreatedAt)}
      </span>
    </div>
  );
}

export function ServiceDetail({ serviceId }: { serviceId: string }) {
  const [service, setService] = useState<Service | null>(null);
  const [deploys, setDeploys] = useState<Deploy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError]     = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setService(null);
    setDeploys([]);
    setError(null);

    Promise.all([api.getService(serviceId), api.listDeploys(serviceId)])
      .then(([svc, deps]) => {
        setService(svc);
        setDeploys(deps ?? []);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "failed to load"))
      .finally(() => setLoading(false));
  }, [serviceId]);

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center text-[12px] text-[#52525b]">
        Loading...
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-[12px] text-[#ef4444]">{error}</div>
      </div>
    );
  }

  if (!service) return null;

  return (
    <div className="flex-1 overflow-y-auto">
      {/* Top bar */}
      <div className="flex items-center justify-between px-6 py-4 border-b border-[#1c1c1f]">
        <div>
          <h1 className="text-[15px] font-semibold text-[#f4f4f5] font-mono">
            {service.name}
          </h1>
          <p className="text-[12px] text-[#52525b] mt-0.5">
            Registered {relativeTime(service.created_at)}
          </p>
        </div>
        {service.active_slot && (
          <div className="flex items-center gap-1.5 text-[12px] text-[#6366f1]">
            <Activity className="w-3.5 h-3.5" />
            <span className="font-mono">{service.active_slot}</span>
          </div>
        )}
      </div>

      <div className="px-6 py-5 flex flex-col gap-5">
        {/* Active deploy banner */}
        <ActiveBanner deploys={deploys} />

        {/* Config panel */}
        <div className="rounded-lg border border-[#1c1c1f] overflow-hidden">
          <div className="px-4 py-2.5 bg-[#111113] border-b border-[#1c1c1f]">
            <span className="text-[11px] font-medium text-[#52525b] uppercase tracking-wider">
              Configuration
            </span>
          </div>
          <div className="px-4 divide-y divide-[#1c1c1f] bg-[#09090b]">
            <InfoRow icon={Globe}     label="Domain"   value={service.domain} />
            <InfoRow icon={GitBranch} label="Repo"     value={service.repo_url} />
            <InfoRow icon={Server}    label="Host"     value={service.host} />
            <InfoRow icon={User}      label="SSH user" value={service.ssh_user} />
            <InfoRow icon={Link}      label="Webhook"  value={service.webhook_url} dim />
          </div>
        </div>

        {/* Deploy history */}
        <DeployList service={service} />
      </div>
    </div>
  );
}
