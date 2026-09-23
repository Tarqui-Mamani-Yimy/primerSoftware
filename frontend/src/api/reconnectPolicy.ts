// Reconnect policy for the diagram realtime WebSocket. Framework-free and
// pure so it can be exercised from scripts/realtime-reconnect-smoke.ts
// without a DOM or test runner. Mirrors mobile/lib/realtime_service.dart
// (commit fcafdfa): growing backoff, no attempt cap, and a backoff reset
// that only happens on a confirmed snapshot — never merely on socket open.

/** Backoff delays in milliseconds: 1s, 2s, 4s, 8s, 16s, then 30s. The last
 * value repeats for every attempt beyond the list, so the wait caps at 30s
 * instead of growing forever or hitting a dead end. */
export const RECONNECT_BACKOFF_MS: readonly number[] = [1000, 2000, 4000, 8000, 16000, 30000];

/** Delay before reconnect attempt number `attempt` (0-based: 0 is the first
 * retry after the initial drop). Clamps to the last backoff step so it
 * repeats indefinitely rather than growing further. */
export function getReconnectDelayMs(attempt: number): number {
  const index = Math.min(Math.max(attempt, 0), RECONNECT_BACKOFF_MS.length - 1);
  return RECONNECT_BACKOFF_MS[index];
}

/** attempt: number of consecutive failed reconnects since the last
 * confirmed snapshot. live: whether the last known state was a confirmed,
 * still-current connection (used to gate immediate-retry triggers). */
export interface ReconnectState {
  readonly attempt: number;
  readonly live: boolean;
}

export const INITIAL_RECONNECT_STATE: ReconnectState = { attempt: 0, live: false };

export type ReconnectEvent =
  | { readonly kind: 'socket-closed' }
  | { readonly kind: 'snapshot-received' }
  | { readonly kind: 'network-online' }
  | { readonly kind: 'document-visible' }
  | { readonly kind: 'ticket-error'; readonly retryable: boolean };

export type ReconnectAction =
  | { readonly kind: 'schedule-retry'; readonly delayMs: number }
  | { readonly kind: 'connect-now' }
  | { readonly kind: 'stop' }
  | { readonly kind: 'noop' };

export interface ReconnectDecision {
  readonly state: ReconnectState;
  readonly action: ReconnectAction;
}

/** Pure decision step for the realtime reconnect state machine. Given the
 * current backoff state and one event, returns the next state and the
 * single action the caller should perform:
 *  - 'snapshot-received' resets backoff and marks the connection live. This
 *    is the ONLY event that resets attempt back to 0.
 *  - 'socket-closed' and a retryable 'ticket-error' schedule the next
 *    attempt with growing backoff. There is no attempt cap.
 *  - a non-retryable 'ticket-error' (auth/membership failure) stops
 *    retrying entirely instead of scheduling another attempt.
 *  - 'network-online' / 'document-visible' request an immediate attempt,
 *    but only when the connection is not already live.
 */
export function nextReconnectAction(state: ReconnectState, event: ReconnectEvent): ReconnectDecision {
  switch (event.kind) {
    case 'snapshot-received':
      return { state: { attempt: 0, live: true }, action: { kind: 'noop' } };

    case 'socket-closed': {
      const delayMs = getReconnectDelayMs(state.attempt);
      return { state: { attempt: state.attempt + 1, live: false }, action: { kind: 'schedule-retry', delayMs } };
    }

    case 'ticket-error': {
      if (!event.retryable) {
        return { state: { ...state, live: false }, action: { kind: 'stop' } };
      }
      const delayMs = getReconnectDelayMs(state.attempt);
      return { state: { attempt: state.attempt + 1, live: false }, action: { kind: 'schedule-retry', delayMs } };
    }

    case 'network-online':
    case 'document-visible':
      if (state.live) return { state, action: { kind: 'noop' } };
      return { state, action: { kind: 'connect-now' } };

    default:
      return { state, action: { kind: 'noop' } };
  }
}
