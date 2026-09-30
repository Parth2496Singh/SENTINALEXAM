// RuntimeCache Manager for local attempt metadata & state bootstrapping

import {
  CachedAttemptRuntime,
  IndexedDbStore,
  dbStore,
} from './indexedDbStore';
import { AttemptRuntime } from './api';

export class RuntimeCache {
  private store: IndexedDbStore;

  constructor(store: IndexedDbStore = dbStore) {
    this.store = store;
  }

  // Persist server-derived AttemptRuntime to IndexedDB
  async saveRuntime(
    runtime: AttemptRuntime,
    serverTimestampMs?: number
  ): Promise<CachedAttemptRuntime> {
    const nowMs = Date.now();
    const perfNow = performance.now();
    const serverTimeMs = serverTimestampMs || nowMs;
    const skewMs = serverTimeMs - nowMs;

    const responsesMap: Record<string, any> = {};
    if (runtime.responses) {
      runtime.responses.forEach((resp) => {
        responsesMap[resp.question_version_id] = resp.value ?? resp.current_value;
      });
    }

    const cached: CachedAttemptRuntime = {
      attempt_id: runtime.attempt.id,
      exam_id: runtime.exam.id,
      candidate_id: runtime.attempt.candidate_id,
      session_id: runtime.session?.id,
      expires_at: runtime.attempt.expires_at,
      duration_seconds: runtime.exam.duration_seconds || (runtime.exam.duration_minutes ? runtime.exam.duration_minutes * 60 : 3600),
      session_epoch: runtime.session?.session_epoch || 1,
      server_time_skew_ms: skewMs,
      monotonic_anchor_ms: perfNow,
      server_anchor_time_ms: serverTimeMs,
      questions: runtime.questions || [],
      responses: responsesMap,
      last_acknowledged_sequence: 0,
      cached_at: nowMs,
    };

    await this.store.saveRuntimeCache(cached);
    return cached;
  }

  // Retrieve cached runtime from IndexedDB
  async getRuntime(attemptId: string): Promise<CachedAttemptRuntime | null> {
    return await this.store.getRuntimeCache(attemptId);
  }

  // Update responses in local cache (optimistic local update)
  async updateCachedResponse(attemptId: string, questionVersionId: string, value: any): Promise<void> {
    const cached = await this.store.getRuntimeCache(attemptId);
    if (!cached) return;

    cached.responses[questionVersionId] = value;
    cached.cached_at = Date.now();
    await this.store.saveRuntimeCache(cached);
  }

  // Clear runtime cache on explicit completion
  async clearRuntime(attemptId: string): Promise<void> {
    await this.store.clearRuntimeCache(attemptId);
  }
}

export const runtimeCache = new RuntimeCache();
