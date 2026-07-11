"use client";

import { useState } from "react";
import { ServiceSidebar } from "@/components/service-sidebar";
import { ServiceDetail } from "@/components/service-detail";

export default function HomePage() {
  const [selectedId, setSelectedId] = useState<string | null>(null);

  return (
    <div className="flex h-screen overflow-hidden">
      <ServiceSidebar selectedId={selectedId} onSelect={setSelectedId} />

      <main className="flex-1 flex flex-col overflow-hidden bg-[#0a0b0d]">
        {selectedId ? (
          <ServiceDetail key={selectedId} serviceId={selectedId} />
        ) : (
          <EmptyState />
        )}
      </main>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex-1 flex flex-col items-center justify-center gap-4 text-center px-8">
      <pre className="text-[#1a2318] text-xs leading-tight select-none hidden sm:block">
        {`
  ██╗      █████╗ ██╗   ██╗███╗   ██╗ ██████╗██╗  ██╗██████╗  █████╗ ██████╗
  ██║     ██╔══██╗██║   ██║████╗  ██║██╔════╝██║  ██║██╔══██╗██╔══██╗██╔══██╗
  ██║     ███████║██║   ██║██╔██╗ ██║██║     ███████║██████╔╝███████║██║  ██║
  ██║     ██╔══██║██║   ██║██║╚██╗██║██║     ██╔══██║██╔═══╝ ██╔══██║██║  ██║
  ███████╗██║  ██║╚██████╔╝██║ ╚████║╚██████╗██║  ██║██║     ██║  ██║██████╔╝
  ╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═══╝ ╚═════╝╚═╝  ╚═╝╚═╝     ╚═╝  ╚═╝╚═════╝
        `}
      </pre>
      <div className="space-y-2">
        <p className="text-[#4a6048] text-sm font-mono">
          $ select a service from the sidebar
          <span className="cursor-blink">_</span>
        </p>
        <p className="text-[#2a3a28] text-xs font-mono">
          zero-downtime blue/green deployments
        </p>
      </div>
    </div>
  );
}
