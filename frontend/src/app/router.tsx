import { createBrowserRouter, Navigate } from "react-router-dom";
import { LoginPage } from "../pages/LoginPage";
import { AdminLayout } from "../components/layout/AdminLayout";
import { CandidateLayout } from "../components/layout/CandidateLayout";

// Admin Pages
import { AdminDashboard } from "../pages/admin/AdminDashboard";
import { ExamsPage } from "../pages/admin/ExamsPage";
import { ExamDetailPage } from "../pages/admin/ExamDetailPage";
import { QuestionBanksPage } from "../pages/admin/QuestionBanksPage";
import { QuestionBankDetailPage } from "../pages/admin/QuestionBankDetailPage";
import { CandidatesPage } from "../pages/admin/CandidatesPage";
import { AttemptsPage } from "../pages/admin/AttemptsPage";

// Candidate Pages
import { CandidateDashboard } from "../pages/candidate/CandidateDashboard";
import { ExamInstructionsPage } from "../pages/candidate/ExamInstructionsPage";
import { AssessmentRunnerPage } from "../pages/candidate/AssessmentRunnerPage";
import { ResultPage } from "../pages/candidate/ResultPage";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <LoginPage />,
  },
  {
    path: "/admin",
    element: <AdminLayout />,
    children: [
      { index: true, element: <AdminDashboard /> },
      { path: "exams", element: <ExamsPage /> },
      { path: "exams/:id", element: <ExamDetailPage /> },
      { path: "question-banks", element: <QuestionBanksPage /> },
      { path: "question-banks/:bankId", element: <QuestionBankDetailPage /> },
      { path: "candidates", element: <CandidatesPage /> },
      { path: "attempts", element: <AttemptsPage /> },
    ],
  },
  {
    path: "/candidate",
    element: <CandidateLayout />,
    children: [
      { index: true, element: <CandidateDashboard /> },
      { path: "exams/:examId/instructions", element: <ExamInstructionsPage /> },
      { path: "attempts/:attemptId/result", element: <ResultPage /> },
    ],
  },
  {
    path: "/candidate/runner/:attemptId",
    element: <AssessmentRunnerPage />,
  },
  {
    path: "*",
    element: <Navigate to="/" replace />,
  },
]);
