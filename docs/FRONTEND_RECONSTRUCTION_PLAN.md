# SentinelExam — Frontend Reconstruction & Integration Plan

## 1. Current Frontend Architecture

The frontend is a Single-Page Application built with **React 18**, **Vite 5**, **React Router DOM 7**, **TanStack React Query v5**, and **TailwindCSS**.

```
[ Browser UI Components ]
         │
         ▼
[ TanStack React Query / Hooks ]
         │
         ▼
[ api.ts (Unified Request Wrapper) ]
         │
         ▼ (Fetch API with Bearer token)
[ Go Backend API (/api/v1) ]
```

### Architectural Deficiencies Identified During Audit
1. **Unprotected Layouts & Premature Queries:** Protected route components (`AdminLayout` and `CandidateLayout`) previously rendered without verifying token validity or role matching. Queries for `/admin` pages (such as `listCandidates`) executed before identity resolution completed, triggering `403 FORBIDDEN` errors from backend RBAC when candidate tokens were active.
2. **Key Parameter Propagation Breakdown:** Unwrapped list responses and undefined item IDs created cascading navigation to `/admin/exams/undefined` or `/candidate/exams/undefined/instructions`, causing React Query to disable requests and rendering "Assessment not available" or "Exam not found" screens.
3. **Stale State & Cache Persistence Across Auth Boundaries:** React Query cache was not purged upon logout. Changing personas or logging out left cached data from prior roles visible in memory.
4. **Dev Auth Re-login Fallback:** An unconditioned `loginAsDevAdmin()` fallback executed whenever stored tokens were cleared, breaking logout functionality by re-authenticating the user immediately.

---

## 2. Inventory of Frontend Codebase

| File / Component | Category | Current Status | Issues Found | Action Needed |
| :--- | :--- | :--- | :--- | :--- |
| `frontend/src/lib/api.ts` | **REAL API INTEGRATION** | PARTIAL | Mixed pagination envelopes, raw vs unwrapped payloads, missing fallback candidate UUID handling in create | **REPAIR** |
| `frontend/src/hooks/useAuth.tsx` | **REAL API INTEGRATION** | BROKEN | Auto-relogin loop on empty token, missing cache purge on logout | **REPAIR** |
| `frontend/src/app/router.tsx` | **REAL API INTEGRATION** | REUSABLE | Lacks explicit Auth Guard wrappers around admin and candidate routes | **REPAIR** |
| `frontend/src/app/providers.tsx` | **REAL API INTEGRATION** | REUSABLE | Valid QueryClient configuration | **KEEP** |
| `frontend/src/components/layout/AdminLayout.tsx` | **REAL API INTEGRATION** | REPAIR | Needs strict `role === "exam_admin"` guard and synchronous logout navigation | **REPAIR** |
| `frontend/src/components/layout/CandidateLayout.tsx` | **REAL API INTEGRATION** | REPAIR | Needs strict `role === "candidate"` guard and synchronous logout navigation | **REPAIR** |
| `frontend/src/pages/LoginPage.tsx` | **REAL API INTEGRATION** | REUSABLE | Persona bypass buttons work, custom token input works | **KEEP** |
| `frontend/src/pages/admin/AdminDashboard.tsx` | **REAL API INTEGRATION** | REPAIR | Live metrics derived from query length; needs role-gated queries | **REPAIR** |
| `frontend/src/pages/admin/ExamsPage.tsx` | **REAL API INTEGRATION** | REPAIR | Needs guaranteed unique key props and creation time display | **REPAIR** |
| `frontend/src/pages/admin/ExamDetailPage.tsx` | **REAL API INTEGRATION** | REPAIR | Needs `bankId`/`examId` valid UUID guards and safe fallback rendering | **REPAIR** |
| `frontend/src/pages/admin/QuestionBanksPage.tsx` | **REAL API INTEGRATION** | REPAIR | Needs normalized bank ID keys and creation time display | **REPAIR** |
| `frontend/src/pages/admin/QuestionBankDetailPage.tsx` | **REAL API INTEGRATION** | REPAIR | Rejects `'undefined'` IDs; needs question version authoring flow | **REPAIR** |
| `frontend/src/pages/admin/CandidatesPage.tsx` | **REAL API INTEGRATION** | REPAIR | Needs UUID auto-generation for candidate registration | **REPAIR** |
| `frontend/src/pages/admin/AttemptsPage.tsx` | **REAL API INTEGRATION** | REPAIR | Derives attempt log from exam list; needs role-gated query | **REPAIR** |
| `frontend/src/pages/candidate/CandidateDashboard.tsx` | **REAL API INTEGRATION** | REPAIR | `listAssignedExams` returns all exams instead of assigned exams; needs filtering | **REPAIR** |
| `frontend/src/pages/candidate/ExamInstructionsPage.tsx` | **REAL API INTEGRATION** | REPAIR | Rendered "Assessment not available" when `examId` was `"undefined"` | **REPAIR** |
| `frontend/src/pages/candidate/AssessmentRunnerPage.tsx` | **REAL API INTEGRATION** | REAL | Real WAL IndexedDB integration & sync; full exam attempt runtime | **KEEP** |
| `frontend/src/pages/candidate/ResultPage.tsx` | **REAL API INTEGRATION** | REAL | Renders attempt score, pass/fail status, and submission breakdown | **KEEP** |

---

## 3. Forensic Investigation of Reported Browser Failures

### Issue A: CandidateDashboard React Key Warning
* **ROOT CAUSE:** `assignedExams.map((exam: Exam)` used `key={exam.id}`. When an exam object returned from backend lacked a populated `id` property (or contained `undefined`), every row received `key={undefined}`, triggering React's list key warning.
* **EVIDENCE:** `CandidateDashboard.tsx:52` stack trace pointing to `table.tsx:93 / Card`.
* **AFFECTED FILE:** `frontend/src/pages/candidate/CandidateDashboard.tsx`
* **CORRECT FIX:** Standardize `id` normalization in `api.listAssignedExams()` and use `key={exam.id ? \`assigned-${exam.id}-${idx}\` : \`assigned-fallback-${idx}\`}`.

### Issue B: QuestionBanksPage React Key Warning
* **ROOT CAUSE:** `filteredBanks.map((bank: QuestionBank)` evaluated `key={bank.id}`. Backend list response returned raw items where `id` was under `bank_id`, causing `bank.id` to evaluate to `undefined`.
* **EVIDENCE:** `QuestionBanksPage.tsx:115 Warning: Each child in a list should have a unique "key" prop.`
* **AFFECTED FILE:** `frontend/src/pages/admin/QuestionBanksPage.tsx` & `frontend/src/lib/api.ts`
* **CORRECT FIX:** Normalize bank items in `listQuestionBanks()` (`id: item.id || item.bank_id`) and construct guaranteed unique keys: `key={bank.id ? \`qb-${bank.id}-${idx}\` : \`qb-fallback-${idx}\`}`.

### Issue C: Navigation to `/admin/exams/undefined` & "Exam Not Found"
* **ROOT CAUSE:** When rendering exam cards or table links, `<Link to={\`/admin/exams/\${exam.id}\`}>` was evaluated when `exam.id` was `undefined`. Clicking the link navigated to `/admin/exams/undefined`. `ExamDetailPage.tsx` received `id = "undefined"`, disarming the query or fetching `GET /api/v1/exams/undefined` (returning 400/404), rendering "Exam not found".
* **EVIDENCE:** Browser address bar `/admin/exams/undefined` and network console `GET /api/v1/exams/undefined 400`.
* **AFFECTED FILE:** `frontend/src/pages/admin/ExamsPage.tsx`, `frontend/src/pages/admin/ExamDetailPage.tsx`, `frontend/src/lib/api.ts`.
* **CORRECT FIX:** Normalize `id` in `api.listExams()`, add `enabled: role === "exam_admin" && !!id && id !== "undefined"` guard in `ExamDetailPage.tsx`, and prevent rendering links with undefined IDs.

### Issue D: Candidate Sees "Assessment Not Available"
* **ROOT CAUSE:**
  1. `CandidateDashboard.tsx` rendered `<Link to={\`/candidate/exams/\${exam.id}/instructions\`}>`. If `exam.id` was missing, it navigated to `/candidate/exams/undefined/instructions`.
  2. `ExamInstructionsPage.tsx` executed `useQuery` with `enabled: !!examId && examId !== "undefined"`. For `examId = "undefined"`, the query was disabled, `exam` remained `undefined`, triggering `if (!exam) return <p>Assessment not available.</p>`.
  3. `api.listAssignedExams()` called `GET /exams` (all exams). If an unassigned or DRAFT exam was rendered, clicking "Start" triggered `api.startAttempt()`, which backend RBAC/Assignment rules rejected with `403 FORBIDDEN` (candidate not assigned).
* **EVIDENCE:** Candidate clicking exam card redirected to instructions page with empty exam state.
* **AFFECTED FILE:** `frontend/src/lib/api.ts`, `frontend/src/pages/candidate/CandidateDashboard.tsx`, `frontend/src/pages/candidate/ExamInstructionsPage.tsx`.
* **CORRECT FIX:** Filter `listAssignedExams()` items to only include `PUBLISHED` or `LIVE` exams that are assigned to the current candidate, and validate `examId` before rendering links.

### Issue E: Creation Buttons Unreliable / Silently Fail
* **ROOT CAUSE:**
  1. For Candidates: Registering a candidate without a 36-character UUID `keycloak_sub` sent `undefined` to `POST /api/v1/candidates`. The backend handler fell back to Admin token claims, attempting to reuse the Admin's user ID instead of creating a new candidate user profile.
  2. For Exams & Question Banks: React Query mutation success handlers invalidated queries, but because `enabled: role === "exam_admin"` was missing or token was uninitialized, the refetched list failed or did not update the UI.
* **EVIDENCE:** Modal closed after submit, but table did not show new candidate or exam.
* **AFFECTED FILE:** `frontend/src/lib/api.ts`, `frontend/src/pages/admin/CandidatesPage.tsx`, `frontend/src/pages/admin/ExamsPage.tsx`.
* **CORRECT FIX:** In `createCandidate()`, auto-generate a valid 36-character UUID (`crypto.randomUUID()`) when `keycloak_sub` is blank. In modal handlers, invalidate queries and display newly created items.

### Issue F: Logout Unreliable / Broken
* **ROOT CAUSE:**
  1. `useAuth.tsx` contained `useEffect(() => { if (!getStoredAuthToken()) loginAsDevAdmin(); })`. Calling `logout()` cleared the token from `localStorage`, but the component re-render immediately re-logged the user back in as Dev Admin.
  2. `logout()` did not invoke `queryClient.clear()`, leaving prior role query caches alive in memory.
  3. `logout()` did not trigger synchronous router redirection to `/`.
* **EVIDENCE:** Clicking Logout icon in header caused a instantaneous screen flicker and left the user on `/admin`.
* **AFFECTED FILE:** `frontend/src/hooks/useAuth.tsx`, `frontend/src/components/layout/AdminLayout.tsx`, `frontend/src/components/layout/CandidateLayout.tsx`.
* **CORRECT FIX:** Introduce an explicit `sentinelexam_logged_out` flag in `useAuth.tsx`. Clear query cache with `queryClient.clear()`, set state to `null`, and navigate to `/` on logout.

---

## 4. Fake Data & Suspicious Metrics Inventory

| Location | Suspicious / Fake Pattern | Root Cause | Replacement Strategy |
| :--- | :--- | :--- | :--- |
| `AdminDashboard.tsx` | Formerly suspected hardcoded 50/50 counts | Derived from `examList.length` and `candidateList.length`. Was returning 0 when queries failed due to 403s. | Keep real query length count; ensure queries succeed with `role === "exam_admin"`. |
| `CandidatesPage.tsx` | Blank `keycloak_sub` reusing Admin claims | `keycloak_sub` sent as `undefined` in `POST /candidates`. | Auto-generate fresh `crypto.randomUUID()` for new candidate registrations. |
| `CandidateDashboard.tsx` | `listAssignedExams` returning unassigned DRAFT exams | `listAssignedExams` simply called `GET /exams`. | Filter returned exams to `PUBLISHED`/`LIVE` status and verify candidate assignment. |
| `api.ts` | Fallback code `EXAM-SYS` | Missing code property on raw exam objects | Populate `code` from backend ID slice. |
| `utils.ts` | Date missing time/seconds | `formatDate` only formatted month/day/year/hour/minute | Updated `formatDate` to include `second: "2-digit"` for exact creation verification. |

---

## 5. Reconstructed Authentication Lifecycle

```
[ User Action: Click Login Persona ]
                │
                ▼
[ loginAsDevAdmin() / loginAsDevCandidate() ]
                │
                ▼
[ Store token in localStorage ('sentinelexam_token') ]
[ Clear logout flag ('sentinelexam_logged_out') ]
[ Update AuthContext state (token, role, user) ]
                │
                ▼
[ Navigate to Protected Route (/admin or /candidate) ]
                │
                ▼
[ Auth Guard Check (token && role === expected) ]
         ├── PASS ──► Render Layout & Enable Query (role === expected)
         └── FAIL ──► Redirect to / or matching role route
                │
                ▼
[ User Action: Click Logout ]
                │
                ▼
[ logout() invoked ]
         ├── Set 'sentinelexam_logged_out' = 'true'
         ├── Remove 'sentinelexam_token' from localStorage
         ├── Invoke queryClient.clear() (purge cached data)
         ├── Reset AuthContext state (token=null, role=null, user=null)
         └── Navigate to '/' (Login Page)
```

### Strict Auth Rules
1. **No Auto-Relogin After Logout:** When `sentinelexam_logged_out` is set, `useEffect` will not trigger `loginAsDevAdmin()`.
2. **No Cache Pollution:** `queryClient.clear()` purges React Query memory on every logout.
3. **No Unauthenticated Query Execution:** All admin/candidate queries have `enabled: role === "exam_admin"` or `enabled: role === "candidate"`.

---

## 6. Reconstructed API Boundary (`api.ts`)

| Core Operation | Frontend Function | Method | Endpoint | Request Payload | Backend Handler | Response Shape | Frontend Normalization |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **List Exams** | `api.listExams()` | GET | `/exams` | None | `ExamHandler.List` | `{ data: [...], meta: {...} }` | Normalizes `id`, `code`, `duration_minutes` |
| **Get Exam** | `api.getExam(id)` | GET | `/exams/{id}` | None | `ExamHandler.Get` | `{ data: {...} }` | Normalizes `code`, `duration_minutes` |
| **Create Exam** | `api.createExam(input)` | POST | `/exams` | `{ title, description, duration_seconds, navigation, scoring }` | `ExamHandler.Create` | `{ data: {...} }` | Returns created `Exam` object |
| **Publish Exam** | `api.publishExam(id)` | POST | `/exams/{id}/publish` | None | `ExamHandler.Publish` | `{ data: {...} }` | Returns updated `Exam` (SCHEDULED) |
| **Start Exam** | `api.startExam(id)` | POST | `/exams/{id}/start` | None | `ExamHandler.Start` | `{ data: {...} }` | Returns updated `Exam` (LIVE) |
| **List Question Banks** | `api.listQuestionBanks()` | GET | `/question-banks` | None | `QBHandler.List` | `{ data: [...], meta: {...} }` | Normalizes `id: item.id \|\| item.bank_id` |
| **Get Question Bank** | `api.getQuestionBank(id)` | GET | `/question-banks/{id}` | None | `QBHandler.Get` | `{ data: {...} }` | Guards against `'undefined'` ID |
| **Create Question Bank**| `api.createQuestionBank(in)`| POST | `/question-banks` | `{ name, description }` | `QBHandler.Create` | `{ data: {...} }` | Returns created `QuestionBank` |
| **Create Question** | `api.createQuestionWithVersion` | POST | `/question-banks/{id}/questions` | `{ type, content, correct_answer, marks }` | `QBHandler.CreateQuestion` | `{ data: { question, version } }` | Returns `{ question, version }` |
| **Publish Question** | `api.publishQuestion(qId)` | POST | `/questions/{qId}/publish` | None | `QuestionHandler.Publish` | `{ data: {...} }` | Returns published `QuestionVersion` |
| **List Candidates** | `api.listCandidates()` | GET | `/candidates` | None | `CandidateHandler.List` | `{ data: [...], meta: {...} }` | Standardizes `first_name`, `last_name` |
| **Register Candidate** | `api.createCandidate(in)` | POST | `/candidates` | `{ keycloak_sub, email, full_name }` | `CandidateHandler.Create` | `{ data: {...} }` | Auto-generates valid UUID if sub blank |
| **Assign Candidate** | `api.assignCandidate(eId, in)`| POST | `/exams/{eId}/assignments` | `{ candidate_id }` | `AssignHandler.Assign` | `{ data: {...} }` | Returns `ExamAssignment` |
| **Start Attempt** | `api.startAttempt(examId)` | POST | `/exams/{examId}/attempts` | None | `AttemptHandler.Start` | `{ data: {...} }` | Returns `ExamAttempt` |
| **Get Attempt Runtime**| `api.getAttemptRuntime(attId)`| GET | `/attempts/{attId}/runtime` | None | `AttemptHandler.Runtime` | `{ data: {...} }` | Returns runtime questions & responses |
| **Save Response** | `api.saveCandidateResponse` | PUT | `/attempts/{attId}/responses/{qvId}` | `{ response_data }` | `AttemptHandler.SaveResponse` | `{ data: {...} }` | Returns updated `CandidateResponse` |
| **Submit Attempt** | `api.submitAttempt(attId)` | POST | `/attempts/{attId}/submit` | None | `AttemptHandler.Submit` | `{ data: {...} }` | Returns finalized `ExamAttempt` |

---

## 7. Real Minimum Product Loop Workflow

### Admin End-to-End Journey
1. **Create Question Bank:** Navigate to `/admin/question-banks` -> Open Modal -> Enter Name & Description -> Submit -> Bank created with UUID.
2. **Author & Publish Question:** Click "View Questions" (`/admin/question-banks/{bankId}`) -> Add Question Prompt & Answers -> Submit -> Auto-publish version to `ACTIVE`.
3. **Create Exam:** Navigate to `/admin/exams` -> Open Modal -> Enter Title, Description, Duration, Passing Marks -> Submit -> Exam created in `DRAFT` status with exact creation timestamp.
4. **Attach Question to Exam:** Click Exam Title (`/admin/exams/{id}`) -> Go to Questions Tab -> Select Published Question Version -> Click Attach.
5. **Publish & Start Exam:** Click "Publish Assessment" (`DRAFT` -> `SCHEDULED`) -> Click "Start Assessment" (`SCHEDULED` -> `LIVE`).
6. **Assign Candidate:** Go to Candidates Tab -> Select Registered Candidate from dropdown -> Click Assign Candidate.

### Candidate End-to-End Journey
1. **Login:** Access `/login` -> Click "Login as Candidate" -> Navigates to `/candidate`.
2. **View Assigned Assessment:** Active assigned exam appears on Candidate Assessment Hub.
3. **Instructions:** Click "Start Assessment" -> Read duration, points, and rules on `/candidate/exams/{examId}/instructions`.
4. **Initialize Attempt:** Click "Begin Assessment" -> Server creates `ACTIVE` attempt -> Navigates to `/candidate/runner/{attemptId}`.
5. **Answer & Sync:** Select answers -> Responses auto-save to IndexedDB WAL and sync to backend API.
6. **Submit:** Click "Submit Assessment" -> Server finalizes attempt and calculates score -> Navigates to `/candidate/attempts/{attemptId}/result`.

### Admin Audit Journey
1. **Review Attempts:** Navigate to `/admin/attempts` or `/admin/exams/{id}` (Attempts tab) -> View candidate UUID, start time, submission status, and score.

---

## 8. Classification of Frontend Areas

| Component / Area | Classification | Justification |
| :--- | :--- | :--- |
| `lib/api.ts` | **REPAIR** | Core request envelope and ID normalizations are accurate; needed candidate UUID fallback and bank ID fixes. |
| `hooks/useAuth.tsx` | **REPAIR** | Dev authentication model is correct; needed explicit logout flag and cache clearing. |
| `app/router.tsx` | **REPAIR** | Route structure matches domain; needs layout route guards. |
| `components/layout/AdminLayout.tsx` | **REPAIR** | Sidebar & header navigation clean; needs `role === "exam_admin"` guard and logout redirect. |
| `components/layout/CandidateLayout.tsx` | **REPAIR** | Portal header clean; needs `role === "candidate"` guard and logout redirect. |
| `pages/LoginPage.tsx` | **KEEP** | Functioning persona bypass and custom token entry. |
| `pages/admin/AdminDashboard.tsx` | **REPAIR** | Real metrics calculation; needs role-gated queries. |
| `pages/admin/ExamsPage.tsx` | **REPAIR** | Table render clean; needs unique key generator and creation timestamp formatting. |
| `pages/admin/ExamDetailPage.tsx` | **REPAIR** | Comprehensive tabs for overview/questions/candidates/attempts; needs `'undefined'` ID guards. |
| `pages/admin/QuestionBanksPage.tsx` | **REPAIR** | Table render clean; needs normalized `id` keys. |
| `pages/admin/QuestionBankDetailPage.tsx` | **REPAIR** | Question authoring dialog working; needs `'undefined'` ID guards. |
| `pages/admin/CandidatesPage.tsx` | **REPAIR** | Directory table working; needs UUID auto-generation for candidate registration. |
| `pages/admin/AttemptsPage.tsx` | **REPAIR** | Derived attempts log working; needs role-gated query. |
| `pages/candidate/CandidateDashboard.tsx` | **REPAIR** | Exam cards working; needs assigned exam filtering and normalized keys. |
| `pages/candidate/ExamInstructionsPage.tsx` | **REPAIR** | Rules display working; needs `'undefined'` ID fallback handling. |
| `pages/candidate/AssessmentRunnerPage.tsx` | **KEEP** | Complete Phase 4 IndexedDB WAL offline persistence and answer runner. |
| `pages/candidate/ResultPage.tsx` | **KEEP** | Complete score breakdown and pass/fail summary. |

---

## 9. Proposed Implementation & Remediation Order

1. **Step 1: Core API & Auth Boundary Stabilization**
   - Verify `api.ts` response unwrapping and ID normalization.
   - Verify `useAuth.tsx` explicit `sentinelexam_logged_out` flag and `queryClient.clear()` on logout.
2. **Step 2: Route Protection & Layout Guards**
   - Enforce `token` and `role` checks in `AdminLayout.tsx` and `CandidateLayout.tsx`.
   - Gate all admin queries with `enabled: role === "exam_admin"`.
   - Gate candidate queries with `enabled: role === "candidate"`.
3. **Step 3: Table Mapping & Key Normalization**
   - Apply guaranteed unique keys (`key={item.id ? \`prefix-${item.id}-${idx}\` : \`prefix-fallback-${idx}\`}`) across all page maps.
4. **Step 4: Creation Form & Candidate UUID Wiring**
   - Ensure `createCandidate()` generates valid UUIDs when `keycloak_sub` is blank.
   - Format exact creation timestamps with seconds (`second: "2-digit"`) in `utils.ts`.
5. **Step 5: Full Browser Verification & Hard Refresh Protocol**
   - Validate live browser workflows for Admin creation, Candidate attempt execution, score display, and logout.

---

## 10. Explicit Unresolved Problems & Next Steps

1. **Browser Environment Playwright Driver Issue:**
   - Automated browser tools (`open_browser_url`) fail due to host environment Playwright binary incompatibility (`%1 is not a valid Win32 application`).
   - *Next Step:* Manual browser verification on `http://localhost:3000` following a hard refresh (`Ctrl + Shift + R`) remains necessary.
2. **Backend Assigned Exams Endpoint:**
   - Currently, `api.listAssignedExams()` calls `GET /api/v1/exams` (which lists all system exams) and relies on frontend filtering.
   - *Next Step:* Filter list items by `PUBLISHED` / `LIVE` status to ensure candidates only see actionable assessments.
