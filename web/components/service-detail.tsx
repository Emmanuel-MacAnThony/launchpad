"use client";

import { useEffect, useState, useCallback } from "react";
import { Globe, GitBranch, Server, Link, User } from "lucide-react";
import { api, type Service } from "@/lib/api";
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

export function ServiceDetail({ serviceId }: { serviceId: string }) {
  const [service, setService] = useState<Service | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError]     = useState<string | null>(null);

  // Refetch without the loading flash — used after a rollback/deploy so the
  // active_slot updates in place while the deploy cards stay mounted.
  const refreshService = useCallback(async () => {
    try {
      const svc = await api.getService(serviceId);
      setService(svc);
    } catch {
      // Keep the last-known service on a transient failure; the next poll retries.
    }
  }, [serviceId]);

  useEffect(() => {
    setLoading(true);
    setService(null);
    setError(null);

    api.getService(serviceId)
      .then((svc) => setService(svc))
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
          <h1 className="text-[20px] font-semibold text-[#f4f4f5] uppercase tracking-wide" style={{ fontFamily: "var(--font-anton)" }}>
            {service.name}
          </h1>
          <p className="text-[12px] text-[#52525b] mt-0.5">
            Registered {relativeTime(service.created_at)}
          </p>
        </div>
        {service.active_slot && (
          <div className="flex items-center gap-2">
            <span className="w-1.5 h-1.5 rounded-full pulse-active"
              style={{ backgroundColor: service.active_slot === "blue" ? "#60a5fa" : "#2dd4bf" }}
            />
            <span className="text-[11px] text-[#52525b] uppercase tracking-wide">Live on</span>
            <SlotBadge slot={service.active_slot} />
          </div>
        )}
      </div>

      <div className="px-6 py-5 flex flex-col gap-5">
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
        <DeployList service={service} onServiceRefresh={refreshService} />
      </div>
    </div>
  );
}
