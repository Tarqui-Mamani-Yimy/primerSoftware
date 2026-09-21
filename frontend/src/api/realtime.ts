import { UMLDiagramDocument } from '../types';
import { getApiBaseUrl } from './diagramApi';

// Authenticated realtime client for the Go presence hub. It mirrors the
// server contract in backend/internal/{domain,realtime}: envelope types,
// snapshot/join/leave/changed/error payloads, the one-shot ?ticket=
// handshake, and the presence.heartbeat keepalive. No cursor sharing,
// no CRDT/OT, no meetings — presence roster plus change awareness only.

export interface PresenceMember {
  userId: string;
  displayName: string;
  lastSeen: string;
}

export interface PresenceSnapshotPayload {
  projectId: string;
  diagramId: string;
  members: PresenceMember[];
  document: UMLDiagramDocument;
  version: number;
  reviewNumber: number;
}

export interface PresenceDeltaPayload {
  projectId: string;
  diagramId: string;
  member: PresenceMember;
}

export interface DiagramChangedEvent {
  actorId: string;
  reviewNumber: number;
  version: number;
  kind: string;
  message?: string;
}

export interface RealtimeErrorPayload {
  code: string;
  message: string;
}

export type RealtimeServerType =
  | 'snapshot'
  | 'presence.join'
  | 'presence.leave'
  | 'diagram.changed'
  | 'error';

export interface RealtimeEnvelope<T = unknown> {
  type: RealtimeServerType | string;
  serverTime: string;
  payload?: T;
}

export interface RealtimeHandlers {
  onSnapshot?: (payload: PresenceSnapshotPayload) => void;
  onJoin?: (payload: PresenceDeltaPayload) => void;
  onLeave?: (payload: PresenceDeltaPayload) => void;
  onChanged?: (payload: DiagramChangedEvent) => void;
  onError?: (payload: RealtimeErrorPayload) => void;
  onClose?: () => void;
}

// Heartbeat every 20s: well inside the server freshness window (60s) so a
// healthy tab is never swept, without chattering every few seconds.
const HEARTBEAT_INTERVAL_MS = 20000;

/** Derives the diagram WebSocket URL from the API base: http→ws, https→wss,
 * path .../projects/{p}/diagrams/{id}/ws?ticket=... */
export function buildRealtimeWsUrl(projectId: string, diagramId: string, ticket: string): string {
  const base = getApiBaseUrl().replace(/\/$/, '');
  const wsBase = base.replace(/^http:/, 'ws:').replace(/^https:/, 'wss:');
  return `${wsBase}/projects/${projectId}/diagrams/${diagramId}/ws?ticket=${encodeURIComponent(ticket)}`;
}

export class RealtimeClient {
  private socket: WebSocket | null = null;
  private heartbeatTimer: ReturnType<typeof setInterval> | undefined;
  private closed = false;

  constructor(
    private readonly url: string,
    private readonly handlers: RealtimeHandlers,
  ) {}

  connect(): void {
    this.closed = false;
    const socket = new WebSocket(this.url);
    this.socket = socket;
    socket.onopen = () => {
      // Disconnected while the handshake was in flight: never start the
      // heartbeat on a dead socket, just close it.
      if (this.closed) {
        try {
          socket.close();
        } catch {
          // Best effort only.
        }
        return;
      }
      this.heartbeatTimer = setInterval(() => {
        if (socket.readyState === WebSocket.OPEN) {
          socket.send(JSON.stringify({ type: 'presence.heartbeat' }));
        }
      }, HEARTBEAT_INTERVAL_MS);
    };
    socket.onmessage = (event) => {
      // Drop events queued before disconnect() so stale sockets never
      // drive callbacks on the wrong document.
      if (this.closed) return;
      let envelope: RealtimeEnvelope;
      try {
        envelope = JSON.parse(String(event.data)) as RealtimeEnvelope;
      } catch {
        return;
      }
      switch (envelope.type) {
        case 'snapshot':
          this.handlers.onSnapshot?.(envelope.payload as PresenceSnapshotPayload);
          break;
        case 'presence.join':
          this.handlers.onJoin?.(envelope.payload as PresenceDeltaPayload);
          break;
        case 'presence.leave':
          this.handlers.onLeave?.(envelope.payload as PresenceDeltaPayload);
          break;
        case 'diagram.changed':
          this.handlers.onChanged?.(envelope.payload as DiagramChangedEvent);
          break;
        case 'error':
          this.handlers.onError?.(envelope.payload as RealtimeErrorPayload);
          break;
        default:
          // Unknown future envelope kinds are dropped so server-side
          // protocol additions never break this client.
          break;
      }
    };
    socket.onclose = () => {
      if (this.heartbeatTimer) {
        clearInterval(this.heartbeatTimer);
        this.heartbeatTimer = undefined;
      }
      this.socket = null;
      if (!this.closed) this.handlers.onClose?.();
    };
  }

  /** Best-effort leave + close: the server also sweeps silent peers, so a
   * dropped leave here is harmless. */
  disconnect(): void {
    this.closed = true;
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = undefined;
    }
    const socket = this.socket;
    this.socket = null;
    if (socket && socket.readyState === WebSocket.OPEN) {
      try {
        socket.send(JSON.stringify({ type: 'presence.leave' }));
      } catch {
        // Best effort only.
      }
    }
    try {
      socket?.close();
    } catch {
      // Best effort only.
    }
  }
}
