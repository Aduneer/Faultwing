export type IssueStatus = "open" | "resolved" | "ignored";

export interface User {
  id: number;
  email: string;
  created_at: string;
}

export interface Session {
  user: User;
  token: string;
  expires_at: string;
}

export interface Project {
  id: number;
  owner_id: number;
  name: string;
  created_at: string;
}

export interface CreatedProject {
  project: Project;
  api_key: string;
}

export interface Issue {
  id: number;
  project_id: number;
  fingerprint: string;
  exception_type: string;
  message: string;
  stacktrace?: string;
  status: IssueStatus;
  event_count: number;
  first_seen: string;
  last_seen: string;
  resolved_at?: string;
  created_at: string;
}

export interface IssueDetail extends Issue {
  environments: string[];
  releases: string[];
}

export interface IssuePage {
  issues: Issue[];
  next_cursor?: string;
}

export class APIError extends Error {
  constructor(public readonly status: number, message: string) {
    super(message);
    this.name = "APIError";
  }
}

// The dashboard is served by Faultwing's origin in production. Keeping requests
// same-origin means sessions never need a browser CORS exception.
const apiBase = "";

async function request<T>(path: string, options: RequestInit = {}, token?: string): Promise<T> {
  const headers = new Headers(options.headers);
  if (options.body) headers.set("Content-Type", "application/json");
  if (token) headers.set("Authorization", `Bearer ${token}`);
  const response = await fetch(`${apiBase}/api/v1${path}`, { ...options, headers });
  if (!response.ok) {
    let message = `Request failed (${response.status})`;
    try {
      const body = await response.json() as { error?: string };
      message = body.error ?? message;
    } catch { /* non-JSON errors still receive a useful fallback */ }
    throw new APIError(response.status, message);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export const faultwing = {
  register: (email: string, password: string) =>
    request<User>("/auth/register", { method: "POST", body: JSON.stringify({ email, password }) }),
  login: (email: string, password: string) =>
    request<Session>("/auth/login", { method: "POST", body: JSON.stringify({ email, password }) }),
  logout: (token: string) => request<void>("/auth/logout", { method: "POST" }, token),
  projects: (token: string) => request<Project[]>("/projects", {}, token),
  createProject: (token: string, name: string) =>
    request<CreatedProject>("/projects", { method: "POST", body: JSON.stringify({ name }) }, token),
  issues: (token: string, projectId: number, status?: IssueStatus, cursor?: string) => {
    const query = new URLSearchParams({ limit: "25" });
    if (status) query.set("status", status);
    if (cursor) query.set("cursor", cursor);
    return request<IssuePage>(`/projects/${projectId}/issues?${query}`, {}, token);
  },
  issue: (token: string, projectId: number, issueId: number) =>
    request<IssueDetail>(`/projects/${projectId}/issues/${issueId}`, {}, token),
  updateIssue: (token: string, projectId: number, issueId: number, status: IssueStatus) =>
    request<Issue>(`/projects/${projectId}/issues/${issueId}`, { method: "PATCH", body: JSON.stringify({ status }) }, token),
  realtimeURL(projectId: number) {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    return `${protocol}//${window.location.host}/api/v1/projects/${projectId}/realtime`;
  },
};
