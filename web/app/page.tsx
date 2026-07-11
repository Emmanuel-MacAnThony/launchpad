"use client";

import { useState } from "react";
import { ServiceSidebar } from "@/components/service-sidebar";
import { ServiceDetail } from "@/components/service-detail";

export default function HomePage() {
  const [selectedId, setSelectedId] = useState<string | null>(null);

  return (
    <div className="flex h-screen overflow-hidden bg-[#09090b]">
      <ServiceSidebar selectedId={selectedId} onSelect={setSelectedId} />

      <main className="flex-1 flex flex-col overflow-hidden bg-grid-dark">
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
    <div className="flex-1 flex flex-col items-center justify-center gap-6 text-center px-8">
      <div className="overflow-hidden hidden sm:block">
      <pre className="ascii-art text-[#6366f1]/[0.08] text-[11px] leading-tight select-none font-mono">
        {`██╗      █████╗ ██╗   ██╗███╗   ██╗ ██████╗██╗  ██╗██████╗  █████╗ ██████╗
██║     ██╔══██╗██║   ██║████╗  ██║██╔════╝██║  ██║██╔══██╗██╔══██╗██╔══██╗
██║     ███████║██║   ██║██╔██╗ ██║██║     ███████║██████╔╝███████║██║  ██║
██║     ██╔══██║██║   ██║██║╚██╗██║██║     ██╔══██║██╔═══╝ ██╔══██║██║  ██║
███████╗██║  ██║╚██████╔╝██║ ╚████║╚██████╗██║  ██║██║     ██║  ██║██████╔╝
╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═══╝ ╚═════╝╚═╝  ╚═╝╚═╝     ╚═╝  ╚═╝╚═════╝`}
      </pre>
      </div>
      <div className="space-y-1.5">
        <p className="text-[14px] text-[#a1a1aa]">
          Select a service from the sidebar
        </p>
        <p className="text-[12px] text-[#52525b]">
          Zero-downtime blue/green deployments
        </p>
      </div>
    </div>
  );
}
