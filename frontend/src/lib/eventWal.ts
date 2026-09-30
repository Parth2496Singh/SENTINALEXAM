// Event WAL Manager & Local Sequence Allocator

import {
  CandidateWALEvent,
  CandidateEventType,
  IndexedDbStore,
  dbStore,
} from './indexedDbStore';

// Generate RFC4122 v4 UUID using crypto.randomUUID if available or fallback
export function generateUUID(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

// Client tab instance ID generated once per tab session
const CLIENT_INSTANCE_ID = generateUUID();

export interface CreateEventParams {
  attemptId: string;
  sessionId: string;
  sessionEpoch: number;
  eventType: CandidateEventType | string;
  payload: Record<string, any>;
  clientTimestamp?: string;
}

export class EventWal {
  private store: IndexedDbStore;
  private currentSequenceMap: Map<string, number> = new Map();

  constructor(store: IndexedDbStore = dbStore) {
    this.store = store;
  }

  // Initialize or re-anchor next local sequence number based on IDB and optional server sequence
  async initSequence(attemptId: string, lastServerSequence: number = 0): Promise<number> {
    const events = await this.store.getWALEventsForAttempt(attemptId);
    let maxLocalSeq = lastServerSequence;

    for (const evt of events) {
      if (evt.sequence_number > maxLocalSeq) {
        maxLocalSeq = evt.sequence_number;
      }
    }

    this.currentSequenceMap.set(attemptId, maxLocalSeq);
    return maxLocalSeq;
  }

  // Get current local sequence for attempt
  getCurrentSequence(attemptId: string): number {
    return this.currentSequenceMap.get(attemptId) || 0;
  }

  // Allocate next sequence number
  private getNextSequence(attemptId: string): number {
    const current = this.currentSequenceMap.get(attemptId) || 0;
    const next = current + 1;
    this.currentSequenceMap.set(attemptId, next);
    return next;
  }

  // Set sequence number (e.g. after server sync or rewind)
  setSequence(attemptId: string, seq: number): void {
    const current = this.currentSequenceMap.get(attemptId) || 0;
    if (seq >= current) {
      this.currentSequenceMap.set(attemptId, seq);
    }
  }

  // Critical Durability Invariant:
  // Create event -> Persist to IndexedDB -> Return created event ONLY after IDB commit
  async recordEvent(params: CreateEventParams): Promise<CandidateWALEvent> {
    const seq = this.getNextSequence(params.attemptId);
    const eventId = generateUUID();

    const event: CandidateWALEvent = {
      event_id: eventId,
      attempt_id: params.attemptId,
      session_id: params.sessionId,
      session_epoch: params.sessionEpoch,
      sequence_number: seq,
      event_type: params.eventType,
      client_instance_id: CLIENT_INSTANCE_ID,
      client_timestamp: params.clientTimestamp || new Date().toISOString(),
      payload: params.payload,
      sync_status: 'PENDING',
      retry_count: 0,
      created_at: Date.now(),
    };

    // Await IDB write before resolving
    await this.store.putWALEvent(event);
    return event;
  }

  // Recovery on startup/restart: recover any stuck SYNCING events back to PENDING
  async recoverOnStartup(attemptId: string): Promise<CandidateWALEvent[]> {
    await this.store.recoverStaleSyncingEvents(attemptId);
    return await this.store.getWALEventsForAttempt(attemptId);
  }

  // Mark PENDING events as SYNCING prior to network transmission batch
  async markEventsSyncing(events: CandidateWALEvent[]): Promise<void> {
    for (const evt of events) {
      if (evt.sync_status === 'PENDING') {
        evt.sync_status = 'SYNCING';
        evt.last_attempted_at = Date.now();
        evt.retry_count = (evt.retry_count || 0) + 1;
        await this.store.putWALEvent(evt);
      }
    }
  }

  // Mark events as ACKNOWLEDGED based on last committed sequence from server
  async markEventsAcknowledged(attemptId: string, lastCommittedSeq: number): Promise<void> {
    const events = await this.store.getWALEventsForAttempt(attemptId);
    for (const evt of events) {
      if (evt.sequence_number <= lastCommittedSeq && evt.sync_status !== 'ACKNOWLEDGED') {
        evt.sync_status = 'ACKNOWLEDGED';
        await this.store.putWALEvent(evt);
      }
    }
  }

  // Prune ACKNOWLEDGED events safely (unacknowledged events are NEVER pruned)
  async pruneAcknowledged(attemptId: string, upToSequence?: number): Promise<number> {
    return await this.store.pruneAcknowledgedEvents(attemptId, upToSequence);
  }

  // Get all unacknowledged (PENDING / SYNCING) events
  async getUnacknowledgedEvents(attemptId: string): Promise<CandidateWALEvent[]> {
    const all = await this.store.getWALEventsForAttempt(attemptId);
    return all.filter((e) => e.sync_status === 'PENDING' || e.sync_status === 'SYNCING');
  }

  // Get pending events ready for dispatch
  async getPendingEvents(attemptId: string): Promise<CandidateWALEvent[]> {
    return await this.store.getWALEventsByStatus(attemptId, 'PENDING');
  }
}

export const eventWal = new EventWal();
