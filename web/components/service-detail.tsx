"use client";

import { useEffect, useState } from "react";
import {
  Server,
  Globe,
  GitBranch,
  Key,
  Link,
  Activity,
  ChevronRight,
} from "lucide-react";
import { api, type Service, type Deploy } from "@/lib/api";
import { DeployList } from "@/components/deploy-list";
import { SlotBadge } from "@/components/status-badge";
import { relativeTime, cn } from "@/lib/utils";

function Row({
  icon: Icon,
  label,
  value,
  mono = true,
  dim = false,
}: {
  icon: React.ElementType;
  label: string;
  value: string;
  mono?: boolean;
  dim?: boolean;
}) {
  return (
    <div className="flex items-start gap-3 py-2 border-b border-[#1a2318] last:border-b-0">
      <Icon className="w-3.5 h-3.5 text-[#4a6048] mt-0.5 shrink-0" />
      <span className="text-[#4a6048] text-xs w-24 shrink-0">{label}</span>
      <span
        className={cn(
          "text-xs break-all",
          mono ? "font-mono" : "",
          dim ? "text-[#4a6048]" : "text-[#a8c8a8]"
        )}
      >
        {value}
      </span>
    </div>
  );
}

function ActiveSlotDisplay({
  deploys,
}: {
  deploys: Deploy[];
}) {
  const activeDeploy = deploys.find((d) => d.Status === "active");

  if (!activeDeploy) {
    return (
      <div className="flex items-center gap-3 px-4 py-3 border border-dashed border-[#1a2318] rounded text-xs text-[#4a6048]">
        <span className="w-2 h-2 rounded-full bg-[#1a2318] inline-block" />
        no active deployment
      </div>
    );
  }

  const slotColor =
    activeDeploy.Slot === "blue" ? "#60a5fa" : "#00ff41";

  return (
    <div
      className="flex items-center justify-between px-4 py-3 border rounded"
      style={{
        borderColor: `${slotColor}30`,
        backgroundColor: `${slotColor}05`,
      }}
    >
      <div className="flex items-center gap-3">
        <Activity className="w-3.5 h-3.5" style={{ color: slotColor }} />
        <div>
          <div className="text-xs text-[#4a6048]">active slot</div>
          <div className="flex items-center gap-2 mt-0.5">
            <SlotBadge slot={activeDeploy.Slot} />
            <ChevronRight className="w-3 h-3 text-[#4a6048]" />
            <code
              className="text-xs"
              style={{ color: slotColor }}
            >
              {activeDeploy.CommitSHA.slice(0, 7)}
            </code>
            <span className="text-xs text-[#4a6048] truncate max-w-[200px]">
              {activeDeploy.CommitMessage}
            </span>
          </div>
        </div>
      </div>
      <span className="text-xs text-[#4a6048]">
        {relativeTime(activeDeploy.FinishedAt ?? activeDeploy.CreatedAt)}
      </span>
    </div>
  );
}

export function ServiceDetail({ serviceId }: { serviceId: string }) {
  const [service, setService] = useState<Service | null>(null);
  const [deploys, setDeploys] = useState<Deploy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
      .catch((e) => {
        setError(e instanceof Error ? e.message : "failed to load service");
      })
      .finally(() => setLoading(false));
  }, [serviceId]);

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center text-xs text-[#4a6048]">
        loading<span className="cursor-blink">_</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-xs text-[#ef4444] font-mono">
          $ error: {error}
        </div>
      </div>
    );
  }

  if (!service) return null;

  return (
    <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-6">
      {/* Service name */}
      <div>
        <div className="flex items-center gap-2 mb-1">
          <span className="text-[#4a6048] text-sm">~/services/</span>
          <h1 className="text-lg font-mono text-[#00ff41]">{service.name}</h1>
          <span className="cursor-blink text-[#00ff41] text-lg">█</span>
        </div>
        <div className="text-xs text-[#4a6048]">
          registered {relativeTime(service.created_at)}
        </div>
      </div>

      {/* Active slot summary */}
      <ActiveSlotDisplay deploys={deploys} />

      {/* Service info */}
      <div className="border border-[#1a2318] rounded overflow-hidden">
        <div className="px-4 py-2 text-[10px] text-[#4a6048] uppercase tracking-widest bg-[#0a0c0a] border-b border-[#1a2318]">
          config
        </div>
        <div className="px-4 divide-y divide-[#1a2318]">
          <Row icon={Globe} label="domain" value={service.domain} />
          <Row icon={GitBranch} label="repo" value={service.repo_url} />
          <Row icon={Server} label="host" value={service.host} />
          <Row icon={Key} label="ssh_user" value={service.ssh_user} />
          <Row
            icon={Link}
            label="webhook"
            value={service.webhook_url}
            dim
          />
        </div>
      </div>

      {/* Deploy list */}
      <DeployList service={service} />
    </div>
  );
}
