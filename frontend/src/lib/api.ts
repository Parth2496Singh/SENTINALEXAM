// SentinelExam Unified API Client

const API_BASE = (import.meta as any).env?.VITE_API_BASE_URL || '/api/v1';

let authToken: string | null = (typeof localStorage !== 'undefined') ? localStorage.getItem('sentinelexam_token') : null;

export function setAuthToken(token: string | null) {
  authToken = token;
  if (typeof localStorage !== 'undefined') {
    if (token) {
      localStorage.setItem('sentinelexam_token', token);
    } else {
      localStorage.removeItem('sentinelexam_token');
    }
  }
}

export function getStoredAuthToken(): string | null {
  return authToken || ((typeof localStorage !== 'undefined') ? localStorage.getItem('sentinelexam_token') : null);
}

export function clearAuthToken() {
  setAuthToken(null);
}

export class ApiError extends Error {
  status: number;
  data: any;
  code?: string;

  constructor(status: number, message: string, data?: any) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;
    this.code = data?.error?.code || data?.code;
  }
}

async function request<T>(method: string, endpoint: string, body?: any): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  const token = getStoredAuthToken();
  if (token) {
    headers['Authorization'] = token.startsWith('Bearer ') ? token : `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${endpoint}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  const contentType = res.headers.get('content-type');
  let data: any = null;
  if (contentType && contentType.includes('application/json')) {
    data = await res.json();
  }

  if (!res.ok) {
    const errorMsg = data?.error?.message || data?.message || `HTTP ${res.status}: ${res.statusText}`;
    throw new ApiError(res.status, errorMsg, data);
  }

  // Extract data if response is wrapped in standard envelope { data: ... }
  if (data && typeof data === 'object' && 'data' in data && data.data !== undefined) {
    return data.data as T;
  }

  return data as T;
}

// ---------------------------------------------------------------------------
// Type Definitions
// ---------------------------------------------------------------------------

export type QuestionType = 'SINGLE_CHOICE' | 'MULTIPLE_CHOICE' | 'TRUE_FALSE' | 'TEXT' | 'NUMERIC';
export type ExamStatus = 'DRAFT' | 'PUBLISHED' | 'SCHEDULED' | 'LIVE' | 'PAUSED' | 'COMPLETED' | 'ARCHIVED';
export type AttemptStatus = 'NOT_STARTED' | 'ACTIVE' | 'IN_PROGRESS' | 'SUBMITTED' | 'EVALUATED' | 'EXPIRED' | 'TERMINATED' | 'FINALIZED';

export interface QuestionBank {
  id: string;
  name: string;
  description: string;
  status: string;
  created_at: string;
  questions?: Question[];
}

export interface QuestionVersion {
  id: string;
  question_id: string;
  version: number;
  type: QuestionType;
  content: any;
  correct_answer?: any;
  explanation?: string;
  marks: number;
  negative_marks: number;
  status: string;
  created_at: string;
}

export interface Question {
  id: string;
  question_bank_id: string;
  status: string;
  current_version_id?: string;
  created_at: string;
  versions?: QuestionVersion[];
}

export interface ExamQuestion {
  id: string;
  exam_id: string;
  question_id: string;
  question_version_id: string;
  section_name?: string;
  position: number;
  marks?: number;
  negative_marks?: number;
  question_version?: QuestionVersion;
}

export interface ExamAssignment {
  id: string;
  exam_id: string;
  candidate_id: string;
  status: string;
  assigned_at: string;
}

export interface ExamAttempt {
  id: string;
  exam_id: string;
  candidate_id: string;
  status: AttemptStatus;
  started_at: string;
  expires_at: string;
  submitted_at?: string;
}

export interface Exam {
  id: string;
  title: string;
  code?: string;
  description: string;
  status: ExamStatus;
  duration_seconds: number;
  duration_minutes?: number;
  pass_marks?: number;
  total_marks?: number;
  instructions?: string;
  created_at: string;
  questions?: ExamQuestion[];
  assignments?: ExamAssignment[];
  attempts?: ExamAttempt[];
}

export interface Candidate {
  id: string;
  user_id?: string;
  keycloak_sub?: string;
  email: string;
  full_name: string;
  first_name?: string;
  last_name?: string;
  created_at: string;
}

export interface RuntimeQuestion {
  id: string;
  exam_id: string;
  position: number;
  question_version_id: string;
  type: QuestionType;
  marks: number;
  negative_marks: number;
  content: any;
  question_version?: {
    id: string;
    question_id: string;
    version: number;
    type: QuestionType;
    content: any;
    marks: number;
    negative_marks: number;
    status: string;
    created_at: string;
  };
}

export interface CandidateResponse {
  id?: string;
  attempt_id: string;
  question_version_id: string;
  current_value?: any;
  value?: any;
  status: string;
  version: number;
  created_at?: string;
  updated_at?: string;
}

export interface AttemptRuntime {
  attempt: ExamAttempt;
  exam: Exam;
  questions: RuntimeQuestion[];
  responses: CandidateResponse[];
  session?: any;
}

export interface Submission {
  id: string;
  attempt_id: string;
  score?: number;
  max_score?: number;
  status: string;
  submitted_at?: string;
  finalized_at?: string;
}

export interface PaginatedResponse<T> {
  data?: T[];
  items?: T[];
  total?: number;
  meta?: {
    page: number;
    per_page: number;
    total: number;
  };
}

// ---------------------------------------------------------------------------
// Input DTOs
// ---------------------------------------------------------------------------

export interface CreateExamInput {
  title: string;
  code?: string;
  description?: string;
  duration_minutes: number;
  pass_marks?: number;
  total_marks?: number;
  instructions?: string;
}

export interface CreateQuestionBankInput {
  name: string;
  description?: string;
}

export interface CreateQuestionWithVersionInput {
  type: QuestionType;
  content: any;
  correct_answer?: any;
  marks?: number;
  negative_marks?: number;
}

export interface AttachQuestionInput {
  question_version_id: string;
  position?: number;
}

export interface AssignCandidateInput {
  candidate_id: string;
}

export interface CreateCandidateInput {
  keycloak_sub?: string;
  email?: string;
  full_name?: string;
  first_name?: string;
  last_name?: string;
}

// ---------------------------------------------------------------------------
// Unified API Object Export
// ---------------------------------------------------------------------------

export const api = {
  // System Info
  getInfo: () => request<{ platform: string; description: string; version: string; status: string }>('GET', '/info'),

  // Exam Operations
  listExams: async () => {
    const res = await request<any>('GET', '/exams');
    const rawItems = res?.data || (Array.isArray(res) ? res : res?.items || []);
    const items = rawItems.map((e: any, index: number) => {
      const examId = e.id || e.exam_id || e.examId || e.ID || `exam-${index}`;
      return {
        ...e,
        id: examId,
        code: e.code || `EXAM-${String(examId).slice(0, 6).toUpperCase() || 'SYS'}`,
        duration_minutes: e.duration_seconds ? Math.round(e.duration_seconds / 60) : e.duration_minutes || 60,
      };
    });
    return {
      items,
      total: res?.meta?.total ?? items.length,
    };
  },
  getExam: async (id: string) => {
    const e = await request<any>('GET', `/exams/${id}`);
    return {
      ...e,
      code: e.code || `EXAM-${e.id?.slice(0, 6)?.toUpperCase() || 'SYS'}`,
      duration_minutes: e.duration_seconds ? Math.round(e.duration_seconds / 60) : e.duration_minutes || 60,
    };
  },
  createExam: (input: CreateExamInput) =>
    request<Exam>('POST', '/exams', {
      title: input.title,
      description: input.description,
      duration_seconds: (input.duration_minutes || 60) * 60,
      navigation: { allow_back: true, allow_skip: true },
      scoring: { pass_marks: input.pass_marks, total_marks: input.total_marks },
    }),
  publishExam: (id: string) => request<Exam>('POST', `/exams/${id}/publish`),
  startExam: (id: string) => request<Exam>('POST', `/exams/${id}/start`),
  pauseExam: (id: string) => request<Exam>('POST', `/exams/${id}/pause`),
  resumeExam: (id: string) => request<Exam>('POST', `/exams/${id}/resume`),
  completeExam: (id: string) => request<Exam>('POST', `/exams/${id}/complete`),
  archiveExam: (id: string) => request<Exam>('POST', `/exams/${id}/archive`),

  // Question Banks
  listQuestionBanks: async () => {
    const res = await request<any>('GET', '/question-banks');
    const rawItems = res?.data || (Array.isArray(res) ? res : res?.items || []);
    const items = rawItems.map((item: any) => ({
      ...item,
      id: item.id || item.bank_id || item.bankId || item.ID,
    }));
    return {
      items,
      total: res?.meta?.total ?? items.length,
    };
  },
  getQuestionBank: (id: string) => {
    if (!id || id === 'undefined') {
      return Promise.reject(new Error('Valid Question Bank ID is required'));
    }
    return request<QuestionBank>('GET', `/question-banks/${id}`);
  },
  createQuestionBank: (input: CreateQuestionBankInput) =>
    request<QuestionBank>('POST', '/question-banks', {
      name: input.name,
      description: input.description || '',
    }),

  // Question Authoring & Publishing
  createQuestionWithVersion: (bankId: string, input: CreateQuestionWithVersionInput) =>
    request<{ question: Question; version: QuestionVersion }>('POST', `/question-banks/${bankId}/questions`, {
      type: input.type,
      content: input.content,
      correct_answer: input.correct_answer,
      marks: input.marks || 10,
      negative_marks: input.negative_marks || 0,
    }),
  getQuestion: (questionId: string) =>
    request<{ question: Question; version: QuestionVersion }>('GET', `/questions/${questionId}`),
  publishQuestion: (questionId: string) =>
    request<QuestionVersion>('POST', `/questions/${questionId}/publish`),
  retireQuestion: (questionId: string) =>
    request<{ status: string }>('POST', `/questions/${questionId}/retire`),

  // Attachments & Assignments
  attachQuestion: (examId: string, input: AttachQuestionInput) =>
    request<ExamQuestion>('POST', `/exams/${examId}/questions`, {
      question_version_id: input.question_version_id,
      position: input.position || 1,
    }),
  assignCandidate: (examId: string, input: AssignCandidateInput) =>
    request<ExamAssignment>('POST', `/exams/${examId}/assignments`, {
      candidate_id: input.candidate_id,
    }),
  revokeAssignment: (examId: string, candidateId: string) =>
    request<{ status: string }>('DELETE', `/exams/${examId}/assignments/${candidateId}`),

  // Candidate Operations
  listCandidates: async () => {
    const res = await request<any>('GET', '/candidates');
    const items = res?.data || (Array.isArray(res) ? res : res?.items || []);
    return {
      items: items.map((c: any) => {
        let firstName = c.first_name || c.FirstName || '';
        let lastName = c.last_name || c.LastName || '';
        const fullName = c.full_name || c.FullName || `${firstName} ${lastName}`.trim();
        if (!firstName && fullName) {
          const parts = fullName.split(' ');
          firstName = parts[0] || '';
          lastName = parts.slice(1).join(' ') || '';
        }
        return {
          ...c,
          id: c.id || c.ID || c.candidate_id || c.CandidateID,
          full_name: fullName,
          first_name: firstName,
          last_name: lastName,
          email: c.email || c.Email,
          created_at: c.created_at || c.CreatedAt,
        };
      }),
      total: res?.meta?.total ?? items.length,
    };
  },
  createCandidate: (input: CreateCandidateInput) => {
    const fullName = input.full_name || `${input.first_name || ''} ${input.last_name || ''}`.trim() || 'Candidate';
    const sub = (input.keycloak_sub && input.keycloak_sub.trim().length === 36)
      ? input.keycloak_sub.trim()
      : (typeof crypto !== 'undefined' && crypto.randomUUID ? crypto.randomUUID() : '00000000-0000-4000-8000-000000000099');
    const email = input.email || `candidate-${sub.slice(0, 8)}@sentinelexam.internal`;
    return request<Candidate>('POST', '/candidates', {
      keycloak_sub: sub,
      email,
      full_name: fullName,
    });
  },
  listAssignedExams: async () => {
    const res = await request<any>('GET', '/candidate/assignments');
    const assignments = res?.data || (Array.isArray(res) ? res : res?.items || []);
    
    // Fetch details for assigned exams in parallel
    const items = await Promise.all(
      assignments.map(async (a: any, index: number) => {
        const examId = a.exam_id || a.examId || a.id;
        try {
          const exam = await request<any>('GET', `/exams/${examId}`);
          return {
            ...exam,
            id: exam.id || examId || `exam-${index}`,
            assignment_id: a.id,
            assignment_status: a.status,
            code: exam.code || `EXAM-${String(examId).slice(0, 6).toUpperCase()}`,
            duration_minutes: exam.duration_seconds ? Math.round(exam.duration_seconds / 60) : exam.duration_minutes || 60,
          };
        } catch {
          return {
            id: examId,
            assignment_id: a.id,
            title: `Assessment ${String(examId).slice(0, 8)}`,
            code: `EXAM-${String(examId).slice(0, 6).toUpperCase()}`,
            status: a.status || 'ASSIGNED',
            duration_minutes: 60,
            pass_marks: 50,
          };
        }
      })
    );

    return {
      items,
      total: res?.meta?.total ?? items.length,
    };
  },
  getAssignedExam: async (examId: string) => {
    const e = await request<any>('GET', `/exams/${examId}`);
    return {
      ...e,
      code: e.code || `EXAM-${e.id?.slice(0, 6)?.toUpperCase() || 'SYS'}`,
      duration_minutes: e.duration_seconds ? Math.round(e.duration_seconds / 60) : e.duration_minutes || 60,
    };
  },
  startAttempt: (examId: string) => request<ExamAttempt>('POST', `/exams/${examId}/attempts`),

  // Candidate Runtime Execution
  getAttemptRuntime: async (attemptId: string) => {
    const runtime = await request<any>('GET', `/attempts/${attemptId}/runtime`);
    if (runtime) {
      if (runtime.responses) {
        runtime.responses = runtime.responses.map((r: any) => ({
          ...r,
          value: r.value ?? r.current_value,
        }));
      }
      if (runtime.questions) {
        runtime.questions = runtime.questions.map((q: any) => ({
          ...q,
          type: q.type || q.question_version?.type || 'SINGLE_CHOICE',
          marks: q.marks ?? q.question_version?.marks ?? 10,
          negative_marks: q.negative_marks ?? q.question_version?.negative_marks ?? 0,
          content: q.content || q.question_version?.content || {},
        }));
      }
    }
    return runtime as AttemptRuntime;
  },
  saveResponse: (attemptId: string, payload: { question_version_id: string; value: any }) =>
    request<CandidateResponse>('PUT', `/attempts/${attemptId}/responses/${payload.question_version_id}`, {
      value: payload.value,
    }),
  createSession: (attemptId: string, payload: { client_info?: any; client_instance_id?: string }) =>
    request<{ id: string }>('POST', `/attempts/${attemptId}/sessions`, {
      client_instance_id: payload.client_instance_id || 'web-client-01',
    }),
  sendHeartbeat: (sessionId: string, payload?: { network_quality?: string }) =>
    request<{ session_id: string; status: string; last_heartbeat_at: string }>('POST', `/sessions/${sessionId}/heartbeat`, payload || {}),
  submitAttempt: (attemptId: string) => request<Submission>('POST', `/attempts/${attemptId}/submit`),
};
