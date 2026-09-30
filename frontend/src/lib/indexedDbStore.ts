// SentinelExam IndexedDB Persistence Abstractions
// Standard IndexedDB wrapper for runtime_cache and event_wal

export const DB_NAME = 'sentinel_exam_db';
export const DB_VERSION = 1;

export const STORE_RUNTIME_CACHE = 'runtime_cache';
export const STORE_EVENT_WAL = 'event_wal';

export type CandidateEventType =
  | 'sentinel.candidate.option_selected'
  | 'sentinel.candidate.text_answered'
  | 'sentinel.candidate.question_navigated'
  | 'sentinel.candidate.review_flag_toggled'
  | 'sentinel.candidate.heartbeat'
  | 'sentinel.candidate.submission_intended';

export type SyncStatus = 'PENDING' | 'SYNCING' | 'ACKNOWLEDGED' | 'REJECTED';

export interface CachedAttemptRuntime {
  attempt_id: string;
  exam_id: string;
  candidate_id?: string;
  session_id?: string;
  expires_at: string;
  duration_seconds: number;
  session_epoch: number;
  server_time_skew_ms: number; // Server Timestamp - Client Date.now() at fetch
  monotonic_anchor_ms: number; // performance.now() recorded at synchronization
  server_anchor_time_ms: number; // Server epoch time corresponding to monotonic_anchor_ms
  questions: any[];
  responses: Record<string, any>; // question_version_id -> value
  last_acknowledged_sequence: number;
  cached_at: number;
}

export interface CandidateWALEvent {
  event_id: string;              // UUID v4
  attempt_id: string;            // UUID
  session_id: string;            // UUID
  session_epoch: number;         // Integer >= 0
  sequence_number: number;       // Monotonic sequence per attempt (1, 2, 3...)
  event_type: CandidateEventType | string;
  client_instance_id: string;    // Browser tab instance UUID
  client_timestamp: string;      // ISO 8601 UTC
  payload: {
    question_version_id?: string;
    value?: any;
    section_name?: string;
    action?: string;
    [key: string]: any;
  };
  sync_status: SyncStatus;
  retry_count: number;
  last_attempted_at?: number;
  error_message?: string;
  created_at: number;            // Local ms timestamp
}

export class IndexedDbStore {
  private dbPromise: Promise<IDBDatabase> | null = null;

  private getDB(): Promise<IDBDatabase> {
    if (this.dbPromise) return this.dbPromise;

    this.dbPromise = new Promise((resolve, reject) => {
      if (typeof indexedDB === 'undefined') {
        reject(new Error('IndexedDB is not supported in this environment'));
        return;
      }

      const request = indexedDB.open(DB_NAME, DB_VERSION);

      request.onupgradeneeded = () => {
        const db = request.result;

        // 1. Create runtime_cache store (Key: attempt_id)
        if (!db.objectStoreNames.contains(STORE_RUNTIME_CACHE)) {
          db.createObjectStore(STORE_RUNTIME_CACHE, { keyPath: 'attempt_id' });
        }

        // 2. Create event_wal store (Key: [attempt_id, sequence_number])
        if (!db.objectStoreNames.contains(STORE_EVENT_WAL)) {
          const walStore = db.createObjectStore(STORE_EVENT_WAL, {
            keyPath: ['attempt_id', 'sequence_number'],
          });

          walStore.createIndex('by_attempt_status', ['attempt_id', 'sync_status'], { unique: false });
          walStore.createIndex('by_attempt_seq', ['attempt_id', 'sequence_number'], { unique: false });
          walStore.createIndex('by_event_id', 'event_id', { unique: true });
        }
      };

      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error);
    });

    return this.dbPromise;
  }

  // --- Runtime Cache operations ---

  async saveRuntimeCache(cache: CachedAttemptRuntime): Promise<void> {
    const db = await this.getDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_RUNTIME_CACHE, 'readwrite');
      const store = tx.objectStore(STORE_RUNTIME_CACHE);
      const req = store.put(cache);
      req.onsuccess = () => resolve();
      req.onerror = () => reject(req.error);
    });
  }

  async getRuntimeCache(attemptId: string): Promise<CachedAttemptRuntime | null> {
    const db = await this.getDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_RUNTIME_CACHE, 'readonly');
      const store = tx.objectStore(STORE_RUNTIME_CACHE);
      const req = store.get(attemptId);
      req.onsuccess = () => resolve(req.result || null);
      req.onerror = () => reject(req.error);
    });
  }

  async clearRuntimeCache(attemptId: string): Promise<void> {
    const db = await this.getDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_RUNTIME_CACHE, 'readwrite');
      const store = tx.objectStore(STORE_RUNTIME_CACHE);
      const req = store.delete(attemptId);
      req.onsuccess = () => resolve();
      req.onerror = () => reject(req.error);
    });
  }

  // --- Event WAL operations ---

  async putWALEvent(event: CandidateWALEvent): Promise<void> {
    const db = await this.getDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_EVENT_WAL, 'readwrite');
      const store = tx.objectStore(STORE_EVENT_WAL);
      const req = store.put(event);
      req.onsuccess = () => resolve();
      req.onerror = () => reject(req.error);
    });
  }

  async getWALEventsForAttempt(attemptId: string): Promise<CandidateWALEvent[]> {
    const db = await this.getDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_EVENT_WAL, 'readonly');
      const store = tx.objectStore(STORE_EVENT_WAL);
      const index = store.index('by_attempt_seq');
      const range = IDBKeyRange.bound([attemptId, 0], [attemptId, Number.MAX_SAFE_INTEGER]);
      const req = index.getAll(range);

      req.onsuccess = () => {
        const events: CandidateWALEvent[] = req.result || [];
        // Sort explicitly by sequence_number ascending
        events.sort((a, b) => a.sequence_number - b.sequence_number);
        resolve(events);
      };
      req.onerror = () => reject(req.error);
    });
  }

  async getWALEventsByStatus(attemptId: string, status: SyncStatus): Promise<CandidateWALEvent[]> {
    const db = await this.getDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_EVENT_WAL, 'readonly');
      const store = tx.objectStore(STORE_EVENT_WAL);
      const index = store.index('by_attempt_status');
      const range = IDBKeyRange.only([attemptId, status]);
      const req = index.getAll(range);

      req.onsuccess = () => {
        const events: CandidateWALEvent[] = req.result || [];
        events.sort((a, b) => a.sequence_number - b.sequence_number);
        resolve(events);
      };
      req.onerror = () => reject(req.error);
    });
  }

  async getWALEventByEventId(eventId: string): Promise<CandidateWALEvent | null> {
    const db = await this.getDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_EVENT_WAL, 'readonly');
      const store = tx.objectStore(STORE_EVENT_WAL);
      const index = store.index('by_event_id');
      const req = index.get(eventId);

      req.onsuccess = () => resolve(req.result || null);
      req.onerror = () => reject(req.error);
    });
  }

  async deleteWALEvent(attemptId: string, sequenceNumber: number): Promise<void> {
    const db = await this.getDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_EVENT_WAL, 'readwrite');
      const store = tx.objectStore(STORE_EVENT_WAL);
      const req = store.delete([attemptId, sequenceNumber]);
      req.onsuccess = () => resolve();
      req.onerror = () => reject(req.error);
    });
  }

  // Crash / Startup Recovery: convert SYNCING -> PENDING while preserving event_id and sequence
  async recoverStaleSyncingEvents(attemptId: string): Promise<number> {
    const db = await this.getDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_EVENT_WAL, 'readwrite');
      const store = tx.objectStore(STORE_EVENT_WAL);
      const index = store.index('by_attempt_status');
      const range = IDBKeyRange.only([attemptId, 'SYNCING']);
      const req = index.getAll(range);

      req.onsuccess = () => {
        const events: CandidateWALEvent[] = req.result || [];
        if (events.length === 0) {
          resolve(0);
          return;
        }

        let updatedCount = 0;
        events.forEach((evt) => {
          evt.sync_status = 'PENDING';
          store.put(evt);
          updatedCount++;
        });

        tx.oncomplete = () => resolve(updatedCount);
        tx.onerror = () => reject(tx.error);
      };

      req.onerror = () => reject(req.error);
    });
  }

  // Prune ACKNOWLEDGED events up to maxSequence (or all if maxSequence not specified)
  async pruneAcknowledgedEvents(attemptId: string, upToSequence?: number): Promise<number> {
    const db = await this.getDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_EVENT_WAL, 'readwrite');
      const store = tx.objectStore(STORE_EVENT_WAL);
      const index = store.index('by_attempt_status');
      const range = IDBKeyRange.only([attemptId, 'ACKNOWLEDGED']);
      const req = index.getAll(range);

      req.onsuccess = () => {
        const events: CandidateWALEvent[] = req.result || [];
        let deletedCount = 0;

        events.forEach((evt) => {
          if (upToSequence === undefined || evt.sequence_number <= upToSequence) {
            store.delete([attemptId, evt.sequence_number]);
            deletedCount++;
          }
        });

        tx.oncomplete = () => resolve(deletedCount);
        tx.onerror = () => reject(tx.error);
      };

      req.onerror = () => reject(req.error);
    });
  }

  async close(): Promise<void> {
    if (this.dbPromise) {
      const db = await this.dbPromise;
      db.close();
      this.dbPromise = null;
    }
  }
}

export const dbStore = new IndexedDbStore();
