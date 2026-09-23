import { getReconnectDelayMs, nextReconnectAction, INITIAL_RECONNECT_STATE, ReconnectState } from '../src/api/reconnectPolicy';

let failures = 0;
function check(label: string, actual: unknown, expected: unknown) {
  const a = JSON.stringify(actual);
  const e = JSON.stringify(expected);
  if (a !== e) {
    failures += 1;
    console.error(`FAIL ${label}\n  expected: ${e}\n  actual:   ${a}`);
  }
}

// --- Backoff delay sequence: 1s, 2s, 4s, 8s, 16s, then 30s repeating. No
// attempt cap — attempt 100 must still resolve to a finite delay (30s),
// never throw or return undefined. ---
check('delay attempt 0', getReconnectDelayMs(0), 1000);
check('delay attempt 1', getReconnectDelayMs(1), 2000);
check('delay attempt 2', getReconnectDelayMs(2), 4000);
check('delay attempt 3', getReconnectDelayMs(3), 8000);
check('delay attempt 4', getReconnectDelayMs(4), 16000);
check('delay attempt 5', getReconnectDelayMs(5), 30000);
check('delay attempt 6 (beyond table, still 30s)', getReconnectDelayMs(6), 30000);
check('delay attempt 100 (no cap failure)', getReconnectDelayMs(100), 30000);
check('delay negative attempt clamps to first step', getReconnectDelayMs(-1), 1000);

// --- Decider: socket-closed schedules growing backoff, unbounded. ---
let state: ReconnectState = INITIAL_RECONNECT_STATE;
const closedDelays: number[] = [];
for (let i = 0; i < 8; i += 1) {
  const decision = nextReconnectAction(state, { kind: 'socket-closed' });
  state = decision.state;
  if (decision.action.kind === 'schedule-retry') closedDelays.push(decision.action.delayMs);
}
check('8 consecutive closes: growing then repeating 30s, no stop', closedDelays, [1000, 2000, 4000, 8000, 16000, 30000, 30000, 30000]);
check('after 8 closes attempt keeps counting (no cap)', state.attempt, 8);
check('after closes, live is false', state.live, false);

// --- Decider: only a snapshot resets backoff (not merely reconnecting). ---
let resetState: ReconnectState = INITIAL_RECONNECT_STATE;
resetState = nextReconnectAction(resetState, { kind: 'socket-closed' }).state; // attempt -> 1
resetState = nextReconnectAction(resetState, { kind: 'socket-closed' }).state; // attempt -> 2
check('attempt grew across closes before any snapshot', resetState.attempt, 2);
const afterSnapshot = nextReconnectAction(resetState, { kind: 'snapshot-received' });
check('snapshot resets attempt to 0', afterSnapshot.state.attempt, 0);
check('snapshot marks live', afterSnapshot.state.live, true);
check('snapshot action is noop (no scheduling)', afterSnapshot.action.kind, 'noop');
const nextCloseAfterReset = nextReconnectAction(afterSnapshot.state, { kind: 'socket-closed' });
check('backoff restarts at 1s after a snapshot reset', nextCloseAfterReset.action, { kind: 'schedule-retry', delayMs: 1000 });

// --- Decider: online/visible trigger an immediate retry only when not live. ---
const notLive: ReconnectState = { attempt: 3, live: false };
const onlineWhileDown = nextReconnectAction(notLive, { kind: 'network-online' });
check('online while not live: connect-now', onlineWhileDown.action, { kind: 'connect-now' });
check('online while not live: attempt preserved (not a reset)', onlineWhileDown.state.attempt, 3);
const visibleWhileDown = nextReconnectAction(notLive, { kind: 'document-visible' });
check('document visible while not live: connect-now', visibleWhileDown.action, { kind: 'connect-now' });

const live: ReconnectState = { attempt: 0, live: true };
const onlineWhileLive = nextReconnectAction(live, { kind: 'network-online' });
check('online while already live: noop', onlineWhileLive.action, { kind: 'noop' });
const visibleWhileLive = nextReconnectAction(live, { kind: 'document-visible' });
check('document visible while already live: noop', visibleWhileLive.action, { kind: 'noop' });

// --- Decider: ticket errors. Non-retryable (401/403/404) stops retrying;
// any other ticket error (network blip, 5xx, timeout) is retried like a
// closed socket. ---
const nonRetryable = nextReconnectAction(INITIAL_RECONNECT_STATE, { kind: 'ticket-error', retryable: false });
check('non-retryable ticket error stops', nonRetryable.action, { kind: 'stop' });
check('non-retryable ticket error marks not live', nonRetryable.state.live, false);
const retryableTicket = nextReconnectAction(INITIAL_RECONNECT_STATE, { kind: 'ticket-error', retryable: true });
check('retryable ticket error schedules backoff like a close', retryableTicket.action, { kind: 'schedule-retry', delayMs: 1000 });
check('retryable ticket error advances attempt', retryableTicket.state.attempt, 1);

if (failures > 0) {
  throw new Error(`realtime reconnect smoke failed: ${failures} check(s) did not match`);
}
console.log('realtime reconnect smoke passed: 24 checks');
