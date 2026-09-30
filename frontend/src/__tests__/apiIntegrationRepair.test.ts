import { describe, it, expect } from 'vitest';
import { api } from '../lib/api';

describe('Frontend Integration Repair Contract Tests', () => {
  it('api.listExams should normalize code property on exam items', async () => {
    // Mock global fetch for GET /exams
    const mockExamsResponse = {
      data: [
        { id: '11111111-2222-3333-4444-555555555555', title: 'Test Exam', duration_seconds: 3600 }
      ],
      meta: { page: 1, per_page: 20, total: 1 }
    };

    const originalFetch = global.fetch;
    global.fetch = async () => ({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => mockExamsResponse,
    } as any);

    try {
      const res = await api.listExams();
      expect(res.items.length).toBe(1);
      expect(res.items[0].code).toBe('EXAM-111111');
      expect(res.items[0].duration_minutes).toBe(60);
    } finally {
      global.fetch = originalFetch;
    }
  });

  it('api.listCandidates should derive full_name and names when backend returns empty optional components', async () => {
    const mockCandidatesResponse = {
      data: [
        { id: 'cand-1', full_name: 'Jane Doe', email: 'jane@example.com' }
      ]
    };

    const originalFetch = global.fetch;
    global.fetch = async () => ({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => mockCandidatesResponse,
    } as any);

    try {
      const res = await api.listCandidates();
      expect(res.items.length).toBe(1);
      expect(res.items[0].first_name).toBe('Jane');
      expect(res.items[0].last_name).toBe('Doe');
    } finally {
      global.fetch = originalFetch;
    }
  });

  it('api.listQuestionBanks should normalize bank IDs and getQuestionBank reject undefined', async () => {
    const mockBanksResponse = {
      data: [
        { bank_id: 'qb-12345678', name: 'Science Bank', description: 'Physics & Chem' }
      ]
    };

    const originalFetch = global.fetch;
    global.fetch = async () => ({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => mockBanksResponse,
    } as any);

    try {
      const res = await api.listQuestionBanks();
      expect(res.items.length).toBe(1);
      expect(res.items[0].id).toBe('qb-12345678');

      await expect(api.getQuestionBank('undefined')).rejects.toThrow('Valid Question Bank ID is required');
    } finally {
      global.fetch = originalFetch;
    }
  });

  it('api.createCandidate should auto-generate valid 36-char keycloak_sub UUID if blank', async () => {
    let sentBody: any = null;
    const originalFetch = global.fetch;
    global.fetch = async (_url: any, options: any) => {
      sentBody = JSON.parse(options.body);
      return {
        ok: true,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ data: { id: 'cand-new', ...sentBody } }),
      } as any;
    };

    try {
      await api.createCandidate({
        first_name: 'Test',
        last_name: 'Candidate',
        email: 'test@example.com',
      });
      expect(sentBody).not.toBeNull();
      expect(sentBody.keycloak_sub).toBeDefined();
      expect(sentBody.keycloak_sub.length).toBe(36);
      expect(sentBody.full_name).toBe('Test Candidate');
    } finally {
      global.fetch = originalFetch;
    }
  });
});
