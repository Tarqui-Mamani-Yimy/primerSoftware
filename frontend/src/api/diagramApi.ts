import { UMLDiagramDocument } from '../types';
const runtimeApiBaseUrl = (globalThis as typeof globalThis & { __UML_API_BASE_URL__?: string }).__UML_API_BASE_URL__;
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || runtimeApiBaseUrl || 'http://localhost:8080/api/v1';
let accessToken: string | undefined;

export interface AuthenticatedUser { accessToken: string; userId: string; displayName: string; email: string; }
export interface AssignedProject { id: string; name: string; description: string; role: 'OWNER' | 'COLLABORATOR'; diagramCount: number; }
export interface CreateProjectInput { name: string; description?: string; }
export interface CreatedProject extends AssignedProject { accessCode: string; }
export interface DiagramSummary { id: string; name: string; updatedAt: string; version: number; }
export interface DiagramVersion { id: string; versionNumber: number; createdAt: string; createdBy: string; message?: string | null; document: UMLDiagramDocument; }
export class ApiError extends Error {
  constructor(public readonly status: number, public readonly payload?: { message?: string; current?: UMLDiagramDocument }) {
    super(payload?.message ?? `API request failed (${status})`);
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const baseHeaders: Record<string, string> = { 'Content-Type': 'application/json' };
  const initHeaders = init?.headers as Record<string, string> | undefined;
  const headers: Record<string, string> = { ...baseHeaders, ...(initHeaders ?? {}) };
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`;
  const response = await fetch(`${API_BASE_URL}${path}`, { ...init, headers });
  if (!response.ok) {
    const body = await response.json().catch(() => null) as { message?: string; current?: UMLDiagramDocument } | null;
    throw new ApiError(response.status, body ?? undefined);
  }
  return response.json() as Promise<T>;
}

function parseContentDisposition(value: string | null): string | undefined {
  if (!value) return undefined;
  const match = /filename="?([^";]+)"?/i.exec(value);
  return match?.[1] ?? undefined;
}

// Binary sibling of `request` for endpoints that answer with a file instead of
// JSON (the JHipster artifact endpoint streams a zip). Shares the module-level
// Bearer token and surfaces failures as ApiError with the backend message.
async function requestFile(path: string, init?: RequestInit): Promise<BackendArtifactResult> {
  const initHeaders = init?.headers as Record<string, string> | undefined;
  const headers: Record<string, string> = { ...(initHeaders ?? {}) };
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`;
  const response = await fetch(`${API_BASE_URL}${path}`, { ...init, headers });
  if (!response.ok) {
    const body = await response.json().catch(() => null) as { message?: string } | null;
    throw new ApiError(response.status, body ?? undefined);
  }
  const filename = parseContentDisposition(response.headers.get('content-disposition')) ?? 'jhipster-backend.zip';
  return { blob: await response.blob(), filename };
}

export const authApi = {
  login: (email: string, password: string) => request<AuthenticatedUser>('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),
  setToken: (token?: string) => { accessToken = token; },
};

export const projectApi = {
  list: () => request<AssignedProject[]>('/projects'),
  create: (input: CreateProjectInput) => request<CreatedProject>('/projects', { method: 'POST', body: JSON.stringify(input) }),
  join: (accessCode: string) => request<AssignedProject>('/projects/join', { method: 'POST', body: JSON.stringify({ accessCode }) }),
};

// Autosave (PUT) writes the working document; each PUT sends the client's
// known baseline as If-Match. POST /checkpoints is the only path that grows
// the version history and stamps the actor as created_by.
export const diagramApi = {
  list: (projectId: string) => request<DiagramSummary[]>(`/projects/${projectId}/diagrams`),
  create: (projectId: string, d: UMLDiagramDocument) => request<UMLDiagramDocument>(`/projects/${projectId}/diagrams`, { method: 'POST', body: JSON.stringify(d) }),
  get: (p: string, id: string) => request<UMLDiagramDocument>(`/projects/${p}/diagrams/${id}`),
  update: (p: string, id: string, d: UMLDiagramDocument) => {
    const headers: Record<string, string> = {};
    if (typeof d.version === 'number') headers['If-Match'] = `"${d.version}"`;
    return request<UMLDiagramDocument>(`/projects/${p}/diagrams/${id}`, { method: 'PUT', body: JSON.stringify(d), headers });
  },
  checkpoint: (p: string, id: string, d: UMLDiagramDocument, message?: string) => {
    const headers: Record<string, string> = {};
    if (typeof d.version === 'number') headers['If-Match'] = `"${d.version}"`;
    if (message && message.trim().length > 0) headers['X-Checkpoint-Message'] = message.trim();
    return request<DiagramVersion>(`/projects/${p}/diagrams/${id}/checkpoints`, { method: 'POST', body: JSON.stringify(d), headers });
  },
  versions: (p: string, id: string) => request<DiagramVersion[]>(`/projects/${p}/diagrams/${id}/versions`),
  restore: (p: string, id: string, v: number) => request<UMLDiagramDocument>(`/projects/${p}/diagrams/${id}/versions/${v}/restore`, { method: 'POST' }),
};

/** Build options for the JHipster artifact endpoint. Any omitted option falls
 * back to the backend defaults (UmlArchitect / com.umlarchitect / maven / jwt). */
export interface ArtifactConfig {
  baseName?: string;
  packageName?: string;
  buildTool?: 'maven' | 'gradle';
  authenticationType?: 'jwt';
}

/** Result of a successful artifact generation: the zip blob plus the exact
 * filename the backend attached (Content-Disposition), so clients can save it
 * without hardcoding a name. */
export interface BackendArtifactResult {
  blob: Blob;
  filename: string;
}

/** The artifact endpoint generates the real Spring Boot/JPA backend on the
 * server (pinned generator-jhipster) and streams it as a zip. */
export const artifactApi = {
  generate: (projectId: string, diagramId: string, document: UMLDiagramDocument, config?: ArtifactConfig) =>
    requestFile(`/projects/${projectId}/diagrams/${diagramId}/artifact`, {
      method: 'POST',
      body: JSON.stringify({ document, config: config ?? undefined }),
    }),
};
