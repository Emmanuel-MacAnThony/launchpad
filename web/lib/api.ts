const BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8090";

export type DeployStatus =
  | "pending"
  | "building"
  | "active"
  | "failed"
  | "rolled_back";

export type Slot = "blue" | "green";

export interface Service {
  id: string;
  name: string;
  repo_url: string;
  domain: string;
  health_check_url: string;
  host: string;
  ssh_user: string;
  webhook_url: string;
  compose_service: string;
  active_slot: Slot | null;
  created_at: string;
}

export interface CreateServiceInput {
  name: string;
  repo_url: string;
  domain: string;
  health_check_url: string;
  webhook_secret: string;
  host: string;
  ssh_user: string;
  ssh_private_key: string;
  blue_port: number;
  green_port: number;
  container_port: number;
  compose_service: string;
}

export interface Deploy {
  ID: string;
  ServiceID: string;
  Slot: Slot | null;
  Status: DeployStatus;
  CommitSHA: string;
  CommitMessage: string;
  PushedAt: string;
  StartedAt: string | null;
  FinishedAt: string | null;
  CreatedAt: string;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json", ...init?.headers },
    ...init,
  });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`${res.status} ${res.statusText}: ${body}`);
  }
  return res.json() as Promise<T>;
}

export const api = {
  listServices: () => request<Service[]>("/services"),

  getService: (id: string) => request<Service>(`/services/${id}`),

  listDeploys: (serviceID: string) =>
    request<Deploy[]>(`/services/${serviceID}/deploys`),

  createService: (input: CreateServiceInput) =>
    request<Service>("/services", {
      method: "POST",
      body: JSON.stringify(input),
    }),

  rollback: (serviceID: string) =>
    request<{ status: string }>(`/services/${serviceID}/rollback`, {
      method: "POST",
    }),
};
