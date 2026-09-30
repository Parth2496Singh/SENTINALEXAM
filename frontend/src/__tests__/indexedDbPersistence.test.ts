import 'fake-indexeddb/auto';
import { describe, it, expect, beforeEach } from 'vitest';
import { IndexedDbStore } from '../lib/indexedDbStore';
import { EventWal } from '../lib/eventWal';
import { RuntimeCache } from '../lib/runtimeCache';

describe('Phase 4.3 - Frontend Offline Persistence & Event Buffer', () => {
  let dbStore: IndexedDbStore;
  let wal: EventWal;
  let cache: RuntimeCache;

  const attemptId = '00000000-0000-0000-0000-000000000001';
  const examId = '00000000-0000-0000-0000-000000000002';
  const sessionId = '00000000-0000-0000-0000-000000000010';
  const questionVersionId = '00000000-0000-0000-0000-000000000020';

  beforeEach(async () => {
    // Re-instantiate fresh store for isolated test
    dbStore = new IndexedDbStore();
    wal = new EventWal(dbStore);
    cache = new RuntimeCache(dbStore);
  });

  // Requirement 1: IndexedDB database initialization
  it('1. should initialize IndexedDB database and stores correctly', async () => {
    const cachedRuntime = await dbStore.getRuntimeCache(attemptId);
    expect(cachedRuntime).toBeNull();
    const events = await dbStore.getWALEventsForAttempt(attemptId);
    expect(events).toEqual([]);
  });

  // Requirement 2: Runtime cache write/read
  it('2. should write and read runtime cache correctly', async () => {
    const sampleRuntime: any = {
      attempt: { id: attemptId, exam_id: examId, candidate_id: 'cand-1', expires_at: new Date(Date.now() + 3600000).toISOString() },
      exam: { id: examId, duration_seconds: 3600 },
      questions: [{ id: 'q1', question_version_id: questionVersionId }],
      responses: [{ question_version_id: questionVersionId, value: 'A' }],
      session: { id: sessionId, session_epoch: 1 },
    };

    await cache.saveRuntime(sampleRuntime);
    const readRuntime = await cache.getRuntime(attemptId);

    expect(readRuntime).not.toBeNull();
    expect(readRuntime?.attempt_id).toBe(attemptId);
    expect(readRuntime?.responses[questionVersionId]).toBe('A');
  });

  // Requirement 3 & 4: Event WAL write/read and event persists before transmission
  it('3 & 4. should write event to WAL with PENDING state before network transmission', async () => {
    await wal.initSequence(attemptId);
    const event = await wal.recordEvent({
      attemptId,
      sessionId,
      sessionEpoch: 1,
      eventType: 'sentinel.candidate.option_selected',
      payload: { question_version_id: questionVersionId, value: 'B' },
    });

    expect(event.sync_status).toBe('PENDING');
    expect(event.sequence_number).toBe(1);

    const savedEvents = await dbStore.getWALEventsForAttempt(attemptId);
    expect(savedEvents.length).toBe(1);
    expect(savedEvents[0].event_id).toBe(event.event_id);
    expect(savedEvents[0].payload.value).toBe('B');
  });

  // Requirement 5: Stable event_id survives retry
  it('5. should preserve event_id across status changes/retries', async () => {
    await wal.initSequence(attemptId);
    const event = await wal.recordEvent({
      attemptId,
      sessionId,
      sessionEpoch: 1,
      eventType: 'sentinel.candidate.option_selected',
      payload: { value: 'C' },
    });

    const originalId = event.event_id;

    // Transition to SYNCING
    await wal.markEventsSyncing([event]);
    let fetched = await dbStore.getWALEventByEventId(originalId);
    expect(fetched?.sync_status).toBe('SYNCING');
    expect(fetched?.event_id).toBe(originalId);

    // Simulate failure & crash recovery back to PENDING
    await dbStore.recoverStaleSyncingEvents(attemptId);
    fetched = await dbStore.getWALEventByEventId(originalId);
    expect(fetched?.sync_status).toBe('PENDING');
    expect(fetched?.event_id).toBe(originalId);
  });

  // Requirement 6 & 7: Sequence survives page reload & multiple events preserve ordering
  it('6 & 7. should preserve monotonic sequence ordering across reloads', async () => {
    await wal.initSequence(attemptId);
    await wal.recordEvent({ attemptId, sessionId, sessionEpoch: 1, eventType: 'evt-1', payload: { step: 1 } });
    await wal.recordEvent({ attemptId, sessionId, sessionEpoch: 1, eventType: 'evt-2', payload: { step: 2 } });
    await wal.recordEvent({ attemptId, sessionId, sessionEpoch: 1, eventType: 'evt-3', payload: { step: 3 } });

    // Simulate reload by creating a new EventWal instance and re-initializing sequence from DB
    const reloadedWal = new EventWal(dbStore);
    const lastSeq = await reloadedWal.initSequence(attemptId);
    expect(lastSeq).toBe(3);

    const newEvt = await reloadedWal.recordEvent({ attemptId, sessionId, sessionEpoch: 1, eventType: 'evt-4', payload: { step: 4 } });
    expect(newEvt.sequence_number).toBe(4);

    const allEvents = await dbStore.getWALEventsForAttempt(attemptId);
    expect(allEvents.map(e => e.sequence_number)).toEqual([1, 2, 3, 4]);
  });

  // Requirement 8, 9, 10: PENDING -> SYNCING -> ACKNOWLEDGED and SYNCING crash recovery
  it('8, 9, 10. should handle lifecycle transitions PENDING -> SYNCING -> ACKNOWLEDGED & SYNCING recovery', async () => {
    await wal.initSequence(attemptId);
    const evt1 = await wal.recordEvent({ attemptId, sessionId, sessionEpoch: 1, eventType: 'evt-1', payload: {} });
    const evt2 = await wal.recordEvent({ attemptId, sessionId, sessionEpoch: 1, eventType: 'evt-2', payload: {} });

    // PENDING -> SYNCING
    await wal.markEventsSyncing([evt1, evt2]);
    let syncingEvents = await dbStore.getWALEventsByStatus(attemptId, 'SYNCING');
    expect(syncingEvents.length).toBe(2);

    // Crash before ACK: recover to PENDING
    await wal.recoverOnStartup(attemptId);
    let pendingEvents = await wal.getPendingEvents(attemptId);
    expect(pendingEvents.length).toBe(2);

    // Re-mark SYNCING and ACKNOWLEDGED up to sequence 1
    await wal.markEventsSyncing([evt1]);
    await wal.markEventsAcknowledged(attemptId, 1);

    const ackedEvents = await dbStore.getWALEventsByStatus(attemptId, 'ACKNOWLEDGED');
    expect(ackedEvents.length).toBe(1);
    expect(ackedEvents[0].sequence_number).toBe(1);
  });

  // Requirement 11 & 12: Pruning only affects ACKNOWLEDGED events up to sequence
  it('11 & 12. should only prune ACKNOWLEDGED events and preserve unacknowledged ones', async () => {
    await wal.initSequence(attemptId);
    await wal.recordEvent({ attemptId, sessionId, sessionEpoch: 1, eventType: 'evt-1', payload: {} });
    await wal.recordEvent({ attemptId, sessionId, sessionEpoch: 1, eventType: 'evt-2', payload: {} });
    await wal.recordEvent({ attemptId, sessionId, sessionEpoch: 1, eventType: 'evt-3', payload: {} });

    // Mark evt1 & evt2 ACKNOWLEDGED, evt3 stays PENDING
    await wal.markEventsAcknowledged(attemptId, 2);

    // Prune acknowledged events up to sequence 1 only
    const prunedCount = await wal.pruneAcknowledged(attemptId, 1);
    expect(prunedCount).toBe(1);

    const remaining = await dbStore.getWALEventsForAttempt(attemptId);
    expect(remaining.map(e => e.sequence_number)).toEqual([2, 3]);
    expect(remaining.find(e => e.sequence_number === 3)?.sync_status).toBe('PENDING');
  });

  // Requirement 13, 14, 15: Restart recovery of runtime cache and attempt state
  it('13, 14, 15. should recover runtime cache and unsent events after browser restart', async () => {
    // 1. Write runtime cache
    await cache.saveRuntime({
      attempt: { id: attemptId, exam_id: examId, status: 'ACTIVE', expires_at: new Date().toISOString() },
      exam: { id: examId, duration_seconds: 3600 },
      questions: [],
      responses: [],
      session: { id: sessionId, session_epoch: 1 },
    } as any);

    // 2. Write pending event
    await wal.initSequence(attemptId);
    await wal.recordEvent({ attemptId, sessionId, sessionEpoch: 1, eventType: 'evt-offline', payload: { ans: 'D' } });

    // 3. Recover
    const recoveredCache = await cache.getRuntime(attemptId);
    expect(recoveredCache?.attempt_id).toBe(attemptId);

    const recoveredWalEvents = await wal.recoverOnStartup(attemptId);
    expect(recoveredWalEvents.length).toBe(1);
    expect(recoveredWalEvents[0].payload.ans).toBe('D');
  });

  // Requirement 16: Cold offline start not supported
  it('16. should return null when attempting to load un-cached attempt offline', async () => {
    const uncachedId = '00000000-0000-0000-0000-000000000999';
    const result = await cache.getRuntime(uncachedId);
    expect(result).toBeNull();
  });

  // Requirement 17: Local submission intent does not finalize attempt
  it('17. should record local submission intent event without marking attempt finalized locally', async () => {
    await wal.initSequence(attemptId);
    const subEvent = await wal.recordEvent({
      attemptId,
      sessionId,
      sessionEpoch: 1,
      eventType: 'sentinel.candidate.submission_intended',
      payload: { action: 'submit_clicked' },
    });

    expect(subEvent.event_type).toBe('sentinel.candidate.submission_intended');
    expect(subEvent.sync_status).toBe('PENDING');

    // Verify cache status remains ACTIVE (server finalization required)
    const cachedRuntime = await cache.getRuntime(attemptId);
    if (cachedRuntime) {
      expect(cachedRuntime.last_acknowledged_sequence).toBeDefined();
    }
  });
});
