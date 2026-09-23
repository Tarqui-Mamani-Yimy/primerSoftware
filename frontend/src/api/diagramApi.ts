import { UMLDiagramDocument } from '../types';
const runtimeApiBaseUrl = (globalThis as typeof globalThis & { __UML_API_BASE_URL__?: string }).__UML_API_BASE_URL__;
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || runtimeApiBaseUrl || 'http://localhost:8080/api/v1';
let accessToken: string | undefined;

export interface AuthenticatedUser { accessToken: string; userId: string; displayName: string; email: string; }
export interface AssignedProject { id: string; name: string; description: string; role: 'OWNER' | 'COLLABORATOR'; diagramCount: number; }
export interface CreateProjectInput { name: string; description?: string; }
export interface CreatedProject extends AssignedProject { accessCode: string; }
export interface DiagramSummary { id: string; name: string; updatedAt: string; version: number; reviewNumber: number; }
export interface DiagramVersion { id: string; versionNumber: number; reviewNumber: number; createdAt: string; createdBy: string; message?: string | null; document: UMLDiagramDocument; }
export class ApiError extends Error {
  constructor(public readonly status: number, public readonly payload?: { message?: string; current?: UMLDiagramDocument }) {
    super(payload?.message ?? `API request failed (${status})`);
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const baseHeaders: Record<string, string> = init?.body instanceof FormData ? {} : { 'Content-Type': 'application/json' };
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
// known baselines as If-Match (version) and X-Diagram-Review (reviewNumber).
// POST /checkpoints is the only path that grows the version history and
// stamps the actor as created_by. A zero reviewNumber is the legacy path:
// the backend falls back to its live baseline, so callers must always pass
// the last baseline they received.
export const diagramApi = {
  list: (projectId: string) => request<DiagramSummary[]>(`/projects/${projectId}/diagrams`),
  create: (projectId: string, d: UMLDiagramDocument) => request<UMLDiagramDocument>(`/projects/${projectId}/diagrams`, { method: 'POST', body: JSON.stringify(d) }),
  get: (p: string, id: string) => request<UMLDiagramDocument>(`/projects/${p}/diagrams/${id}`),
  update: (p: string, id: string, d: UMLDiagramDocument) => {
    const headers: Record<string, string> = {};
    if (typeof d.version === 'number') headers['If-Match'] = `"${d.version}"`;
    if (typeof d.reviewNumber === 'number') headers['X-Diagram-Review'] = `${d.reviewNumber}`;
    return request<UMLDiagramDocument>(`/projects/${p}/diagrams/${id}`, { method: 'PUT', body: JSON.stringify(d), headers });
  },
  checkpoint: (p: string, id: string, d: UMLDiagramDocument, message?: string) => {
    const headers: Record<string, string> = {};
    if (typeof d.version === 'number') headers['If-Match'] = `"${d.version}"`;
    if (typeof d.reviewNumber === 'number') headers['X-Diagram-Review'] = `${d.reviewNumber}`;
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

/** Base URL of the Go API (without trailing slash). Exported so the
 * realtime client can derive the WebSocket URL from the same source. */
export function getApiBaseUrl(): string {
  return API_BASE_URL;
}

/** Short-lived signed ticket that opens the diagram WebSocket without a
 * custom Authorization header: POST .../realtime-tickets (bearer-gated),
 * then GET .../ws?ticket=... The ticket is one-shot and bound to the
 * project/diagram/user triple. */
export interface RealtimeTicket { ticket: string; expiresIn: number; }
export const realtimeApi = {
  ticket: (p: string, id: string) => request<RealtimeTicket>(`/projects/${p}/diagrams/${id}/realtime-tickets`, { method: 'POST' }),
};

/** The artifact endpoint generates the real Spring Boot/JPA backend on the
 * server (pinned generator-jhipster) and streams it as a zip. */
export const imageImportApi = {
  import: (projectId: string, diagramId: string, file: File): Promise<UMLDiagramDocument> => {
    const form = new FormData();
    form.append('image', file);
    return request<UMLDiagramDocument>(`/projects/${projectId}/diagrams/${diagramId}/import-image`, { method: 'POST', body: form, headers: {} });
  },
};

export interface VoiceTranscription { text: string; }
export const voiceApi = {
  transcribe: (audio: Blob) => request<VoiceTranscription>('/voice/transcriptions', {
    method: 'POST',
    body: audio,
    headers: { 'Content-Type': audio.type || 'audio/webm' },
  }),
};

export const artifactApi = {
  generate: (projectId: string, diagramId: string, document: UMLDiagramDocument, config?: ArtifactConfig) =>
    requestFile(`/projects/${projectId}/diagrams/${diagramId}/artifact`, {
      method: 'POST',
      body: JSON.stringify({ document, config: config ?? undefined }),
    }),
};
