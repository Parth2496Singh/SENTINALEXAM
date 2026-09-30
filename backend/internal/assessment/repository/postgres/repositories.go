// Package postgres implements the repository interfaces against PostgreSQL using pgx.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentinelexam/backend/internal/assessment/domain"
	"github.com/sentinelexam/backend/internal/assessment/models"
	"github.com/sentinelexam/backend/internal/assessment/repository"
)

// DB wraps a pgxpool.Pool and provides construction helpers for all repos.
type DB struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *DB { return &DB{pool: pool} }

// Repositories returns a fully wired repository bundle.
func (d *DB) Repositories() repository.Repositories {
	return repository.Repositories{
		Users:            &userRepo{d.pool},
		Candidates:       &candidateRepo{d.pool},
		QuestionBanks:    &questionBankRepo{d.pool},
		Questions:        &questionRepo{d.pool},
		QuestionVersions: &questionVersionRepo{d.pool},
		Exams:            &examRepo{d.pool},
		ExamQuestions:    &examQuestionRepo{d.pool},
		Assignments:      &assignmentRepo{d.pool},
		Attempts:         &attemptRepo{d.pool},
		Sessions:         &sessionRepo{d.pool},
		Responses:        &responseRepo{d.pool},
		Submissions:      &submissionRepo{d.pool},
		CandidateEvents:  NewCandidateEventRepo(d.pool),
		OutboxEvents:     NewOutboxRepo(d.pool),
	}
}

// notFound converts pgx.ErrNoRows to domain.ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

// ---------------------------------------------------------------------------
// User repository
// ---------------------------------------------------------------------------

type userRepo struct{ pool *pgxpool.Pool }

func (r *userRepo) FindByKeycloakSub(ctx context.Context, sub domain.ID) (*models.User, error) {
	u := &models.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, keycloak_sub, email, display_name, roles, created_at, updated_at
		 FROM users WHERE keycloak_sub = $1`, sub).
		Scan(&u.ID, &u.KeycloakSub, &u.Email, &u.DisplayName, &u.Roles, &u.CreatedAt, &u.UpdatedAt)
	return u, notFound(err)
}

func (r *userRepo) Upsert(ctx context.Context, user *models.User) error {
	if user.ID == uuid.Nil {
		user.ID = domain.NewID()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, keycloak_sub, email, display_name, roles)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (keycloak_sub) DO UPDATE
		 SET email = EXCLUDED.email, display_name = EXCLUDED.display_name,
		     roles = EXCLUDED.roles, updated_at = NOW()`,
		user.ID, user.KeycloakSub, user.Email, user.DisplayName, user.Roles)
	return err
}

// ---------------------------------------------------------------------------
// Candidate repository
// ---------------------------------------------------------------------------

type candidateRepo struct{ pool *pgxpool.Pool }

func (r *candidateRepo) FindByID(ctx context.Context, id domain.ID) (*models.Candidate, error) {
	c := &models.Candidate{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, status, full_name, metadata, created_at, updated_at
		 FROM candidates WHERE id = $1`, id).
		Scan(&c.ID, &c.UserID, &c.Status, &c.FullName, &c.Metadata, &c.CreatedAt, &c.UpdatedAt)
	return c, notFound(err)
}

func (r *candidateRepo) FindByUserID(ctx context.Context, userID domain.ID) (*models.Candidate, error) {
	c := &models.Candidate{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, status, full_name, metadata, created_at, updated_at
		 FROM candidates WHERE user_id = $1`, userID).
		Scan(&c.ID, &c.UserID, &c.Status, &c.FullName, &c.Metadata, &c.CreatedAt, &c.UpdatedAt)
	return c, notFound(err)
}

func (r *candidateRepo) List(ctx context.Context, p domain.Pagination) ([]models.Candidate, int, error) {
	var total int
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM candidates`).Scan(&total)
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, status, full_name, metadata, created_at, updated_at
		 FROM candidates ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		p.Limit(), p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.Candidate
	for rows.Next() {
		var c models.Candidate
		if err := rows.Scan(&c.ID, &c.UserID, &c.Status, &c.FullName, &c.Metadata, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

func (r *candidateRepo) Create(ctx context.Context, c *models.Candidate) error {
	if c.ID == uuid.Nil {
		c.ID = domain.NewID()
	}
	now := time.Now().UTC()
	c.CreatedAt, c.UpdatedAt = now, now
	_, err := r.pool.Exec(ctx,
		`INSERT INTO candidates (id, user_id, status, full_name, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		c.ID, c.UserID, c.Status, c.FullName, c.Metadata, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *candidateRepo) Update(ctx context.Context, c *models.Candidate) error {
	c.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE candidates SET full_name=$2, status=$3, metadata=$4, updated_at=$5 WHERE id=$1`,
		c.ID, c.FullName, c.Status, c.Metadata, c.UpdatedAt)
	return err
}

// ---------------------------------------------------------------------------
// QuestionBank repository
// ---------------------------------------------------------------------------

type questionBankRepo struct{ pool *pgxpool.Pool }

func (r *questionBankRepo) FindByID(ctx context.Context, id domain.ID) (*models.QuestionBank, error) {
	qb := &models.QuestionBank{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, status, created_by, created_at, updated_at
		 FROM question_banks WHERE id = $1`, id).
		Scan(&qb.ID, &qb.Name, &qb.Description, &qb.Status, &qb.CreatedBy, &qb.CreatedAt, &qb.UpdatedAt)
	return qb, notFound(err)
}

func (r *questionBankRepo) List(ctx context.Context, p domain.Pagination) ([]models.QuestionBank, int, error) {
	var total int
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM question_banks`).Scan(&total)
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, status, created_by, created_at, updated_at
		 FROM question_banks ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		p.Limit(), p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.QuestionBank
	for rows.Next() {
		var qb models.QuestionBank
		if err := rows.Scan(&qb.ID, &qb.Name, &qb.Description, &qb.Status, &qb.CreatedBy, &qb.CreatedAt, &qb.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, qb)
	}
	return out, total, rows.Err()
}

func (r *questionBankRepo) Create(ctx context.Context, qb *models.QuestionBank) error {
	if qb.ID == uuid.Nil {
		qb.ID = domain.NewID()
	}
	now := time.Now().UTC()
	qb.CreatedAt, qb.UpdatedAt = now, now
	_, err := r.pool.Exec(ctx,
		`INSERT INTO question_banks (id, name, description, status, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		qb.ID, qb.Name, qb.Description, qb.Status, qb.CreatedBy, qb.CreatedAt, qb.UpdatedAt)
	return err
}

// ---------------------------------------------------------------------------
// Question repository
// ---------------------------------------------------------------------------

type questionRepo struct{ pool *pgxpool.Pool }

func (r *questionRepo) FindByID(ctx context.Context, id domain.ID) (*models.Question, error) {
	q := &models.Question{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, question_bank_id, status, current_version_id, created_by, created_at, updated_at
		 FROM questions WHERE id = $1`, id).
		Scan(&q.ID, &q.QuestionBankID, &q.Status, &q.CurrentVersionID, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt)
	return q, notFound(err)
}

func (r *questionRepo) ListByBank(ctx context.Context, bankID domain.ID, p domain.Pagination) ([]models.Question, int, error) {
	var total int
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM questions WHERE question_bank_id = $1`, bankID).Scan(&total)
	rows, err := r.pool.Query(ctx,
		`SELECT id, question_bank_id, status, current_version_id, created_by, created_at, updated_at
		 FROM questions WHERE question_bank_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		bankID, p.Limit(), p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.Question
	for rows.Next() {
		var q models.Question
		if err := rows.Scan(&q.ID, &q.QuestionBankID, &q.Status, &q.CurrentVersionID, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, q)
	}
	return out, total, rows.Err()
}

func (r *questionRepo) Create(ctx context.Context, q *models.Question) error {
	if q.ID == uuid.Nil {
		q.ID = domain.NewID()
	}
	now := time.Now().UTC()
	q.CreatedAt, q.UpdatedAt = now, now
	_, err := r.pool.Exec(ctx,
		`INSERT INTO questions (id, question_bank_id, status, current_version_id, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		q.ID, q.QuestionBankID, q.Status, q.CurrentVersionID, q.CreatedBy, q.CreatedAt, q.UpdatedAt)
	return err
}

func (r *questionRepo) Update(ctx context.Context, q *models.Question) error {
	q.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE questions SET status=$2, current_version_id=$3, updated_at=$4 WHERE id=$1`,
		q.ID, q.Status, q.CurrentVersionID, q.UpdatedAt)
	return err
}

// ---------------------------------------------------------------------------
// QuestionVersion repository
// ---------------------------------------------------------------------------

type questionVersionRepo struct{ pool *pgxpool.Pool }

func (r *questionVersionRepo) FindByID(ctx context.Context, id domain.ID) (*models.QuestionVersion, error) {
	qv := &models.QuestionVersion{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, question_id, version, type, content, correct_answer, marks, negative_marks, status, created_by, created_at
		 FROM question_versions WHERE id = $1`, id).
		Scan(&qv.ID, &qv.QuestionID, &qv.Version, &qv.Type, &qv.Content, &qv.CorrectAnswer,
			&qv.Marks, &qv.NegativeMarks, &qv.Status, &qv.CreatedBy, &qv.CreatedAt)
	return qv, notFound(err)
}

func (r *questionVersionRepo) FindLatestByQuestion(ctx context.Context, questionID domain.ID) (*models.QuestionVersion, error) {
	qv := &models.QuestionVersion{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, question_id, version, type, content, correct_answer, marks, negative_marks, status, created_by, created_at
		 FROM question_versions WHERE question_id = $1 ORDER BY version DESC LIMIT 1`, questionID).
		Scan(&qv.ID, &qv.QuestionID, &qv.Version, &qv.Type, &qv.Content, &qv.CorrectAnswer,
			&qv.Marks, &qv.NegativeMarks, &qv.Status, &qv.CreatedBy, &qv.CreatedAt)
	return qv, notFound(err)
}

func (r *questionVersionRepo) ListByQuestion(ctx context.Context, questionID domain.ID) ([]models.QuestionVersion, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, question_id, version, type, content, correct_answer, marks, negative_marks, status, created_by, created_at
		 FROM question_versions WHERE question_id = $1 ORDER BY version DESC`, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.QuestionVersion
	for rows.Next() {
		var qv models.QuestionVersion
		if err := rows.Scan(&qv.ID, &qv.QuestionID, &qv.Version, &qv.Type, &qv.Content, &qv.CorrectAnswer,
			&qv.Marks, &qv.NegativeMarks, &qv.Status, &qv.CreatedBy, &qv.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, qv)
	}
	return out, rows.Err()
}

func (r *questionVersionRepo) Create(ctx context.Context, qv *models.QuestionVersion) error {
	if qv.ID == uuid.Nil {
		qv.ID = domain.NewID()
	}
	qv.CreatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO question_versions (id, question_id, version, type, content, correct_answer, marks, negative_marks, status, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		qv.ID, qv.QuestionID, qv.Version, qv.Type, qv.Content, qv.CorrectAnswer,
		qv.Marks, qv.NegativeMarks, qv.Status, qv.CreatedBy, qv.CreatedAt)
	return err
}

func (r *questionVersionRepo) UpdateStatus(ctx context.Context, id domain.ID, status domain.QuestionStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE question_versions SET status=$2 WHERE id=$1`, id, status)
	return err
}

// ---------------------------------------------------------------------------
// Exam repository
// ---------------------------------------------------------------------------

type examRepo struct{ pool *pgxpool.Pool }

func (r *examRepo) FindByID(ctx context.Context, id domain.ID) (*models.Exam, error) {
	e := &models.Exam{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, title, description, status, duration_seconds, scheduled_start, scheduled_end,
		        max_attempts, navigation, scoring, version, created_by, created_at, updated_at
		 FROM exams WHERE id = $1`, id).
		Scan(&e.ID, &e.Title, &e.Description, &e.Status, &e.DurationSeconds,
			&e.ScheduledStart, &e.ScheduledEnd, &e.MaxAttempts,
			&e.Navigation, &e.Scoring, &e.Version, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	return e, notFound(err)
}

func (r *examRepo) List(ctx context.Context, p domain.Pagination) ([]models.Exam, int, error) {
	var total int
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM exams`).Scan(&total)
	rows, err := r.pool.Query(ctx,
		`SELECT id, title, description, status, duration_seconds, scheduled_start, scheduled_end,
		        max_attempts, navigation, scoring, version, created_by, created_at, updated_at
		 FROM exams ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		p.Limit(), p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.Exam
	for rows.Next() {
		var e models.Exam
		if err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.Status, &e.DurationSeconds,
			&e.ScheduledStart, &e.ScheduledEnd, &e.MaxAttempts,
			&e.Navigation, &e.Scoring, &e.Version, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

func (r *examRepo) Create(ctx context.Context, e *models.Exam) error {
	if e.ID == uuid.Nil {
		e.ID = domain.NewID()
	}
	now := time.Now().UTC()
	e.CreatedAt, e.UpdatedAt = now, now
	_, err := r.pool.Exec(ctx,
		`INSERT INTO exams (id, title, description, status, duration_seconds, scheduled_start, scheduled_end,
		                    max_attempts, navigation, scoring, version, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		e.ID, e.Title, e.Description, e.Status, e.DurationSeconds,
		e.ScheduledStart, e.ScheduledEnd, e.MaxAttempts,
		e.Navigation, e.Scoring, e.Version, e.CreatedBy, e.CreatedAt, e.UpdatedAt)
	return err
}

func (r *examRepo) Update(ctx context.Context, e *models.Exam) error {
	e.UpdatedAt = time.Now().UTC()
	e.Version++
	_, err := r.pool.Exec(ctx,
		`UPDATE exams SET title=$2, description=$3, duration_seconds=$4, scheduled_start=$5,
		                  scheduled_end=$6, max_attempts=$7, navigation=$8, scoring=$9,
		                  version=$10, updated_at=$11
		 WHERE id=$1 AND status='DRAFT'`,
		e.ID, e.Title, e.Description, e.DurationSeconds, e.ScheduledStart,
		e.ScheduledEnd, e.MaxAttempts, e.Navigation, e.Scoring, e.Version, e.UpdatedAt)
	return err
}

func (r *examRepo) UpdateStatus(ctx context.Context, id domain.ID, status domain.ExamStatus, version int) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE exams SET status=$2, version=$3, updated_at=NOW() WHERE id=$1`,
		id, status, version+1)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: exam not found or version conflict", domain.ErrConflict)
	}
	return nil
}

// ---------------------------------------------------------------------------
// ExamQuestion repository
// ---------------------------------------------------------------------------

type examQuestionRepo struct{ pool *pgxpool.Pool }

func (r *examQuestionRepo) FindByExam(ctx context.Context, examID domain.ID) ([]models.ExamQuestionDetail, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT eq.id, eq.exam_id, eq.question_version_id, eq.position, eq.created_at,
		        qv.id, qv.question_id, qv.version, qv.type, qv.content, qv.correct_answer,
		        qv.marks, qv.negative_marks, qv.status, qv.created_by, qv.created_at
		 FROM exam_questions eq
		 JOIN question_versions qv ON qv.id = eq.question_version_id
		 WHERE eq.exam_id = $1 ORDER BY eq.position`, examID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ExamQuestionDetail
	for rows.Next() {
		var d models.ExamQuestionDetail
		if err := rows.Scan(
			&d.ID, &d.ExamID, &d.QuestionVersionID, &d.Position, &d.CreatedAt,
			&d.QuestionVersion.ID, &d.QuestionVersion.QuestionID, &d.QuestionVersion.Version,
			&d.QuestionVersion.Type, &d.QuestionVersion.Content, &d.QuestionVersion.CorrectAnswer,
			&d.QuestionVersion.Marks, &d.QuestionVersion.NegativeMarks,
			&d.QuestionVersion.Status, &d.QuestionVersion.CreatedBy, &d.QuestionVersion.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *examQuestionRepo) Add(ctx context.Context, eq *models.ExamQuestion) error {
	if eq.ID == uuid.Nil {
		eq.ID = domain.NewID()
	}
	eq.CreatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO exam_questions (id, exam_id, question_version_id, position, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		eq.ID, eq.ExamID, eq.QuestionVersionID, eq.Position, eq.CreatedAt)
	return err
}

func (r *examQuestionRepo) Remove(ctx context.Context, examID, questionVersionID domain.ID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM exam_questions WHERE exam_id=$1 AND question_version_id=$2`,
		examID, questionVersionID)
	return err
}

func (r *examQuestionRepo) CountByExam(ctx context.Context, examID domain.ID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM exam_questions WHERE exam_id=$1`, examID).Scan(&n)
	return n, err
}

// ---------------------------------------------------------------------------
// Assignment repository
// ---------------------------------------------------------------------------

type assignmentRepo struct{ pool *pgxpool.Pool }

func (r *assignmentRepo) FindByExamAndCandidate(ctx context.Context, examID, candidateID domain.ID) (*models.ExamAssignment, error) {
	a := &models.ExamAssignment{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, exam_id, candidate_id, status, assigned_by, assigned_at, revoked_at
		 FROM exam_assignments WHERE exam_id=$1 AND candidate_id=$2`, examID, candidateID).
		Scan(&a.ID, &a.ExamID, &a.CandidateID, &a.Status, &a.AssignedBy, &a.AssignedAt, &a.RevokedAt)
	return a, notFound(err)
}

func (r *assignmentRepo) FindByID(ctx context.Context, id domain.ID) (*models.ExamAssignment, error) {
	a := &models.ExamAssignment{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, exam_id, candidate_id, status, assigned_by, assigned_at, revoked_at
		 FROM exam_assignments WHERE id=$1`, id).
		Scan(&a.ID, &a.ExamID, &a.CandidateID, &a.Status, &a.AssignedBy, &a.AssignedAt, &a.RevokedAt)
	return a, notFound(err)
}

func (r *assignmentRepo) ListByExam(ctx context.Context, examID domain.ID, p domain.Pagination) ([]models.ExamAssignment, int, error) {
	var total int
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM exam_assignments WHERE exam_id=$1`, examID).Scan(&total)
	rows, err := r.pool.Query(ctx,
		`SELECT id, exam_id, candidate_id, status, assigned_by, assigned_at, revoked_at
		 FROM exam_assignments WHERE exam_id=$1 ORDER BY assigned_at DESC LIMIT $2 OFFSET $3`,
		examID, p.Limit(), p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.ExamAssignment
	for rows.Next() {
		var a models.ExamAssignment
		if err := rows.Scan(&a.ID, &a.ExamID, &a.CandidateID, &a.Status, &a.AssignedBy, &a.AssignedAt, &a.RevokedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (r *assignmentRepo) ListByCandidate(ctx context.Context, candidateID domain.ID, p domain.Pagination) ([]models.ExamAssignment, int, error) {
	var total int
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM exam_assignments WHERE candidate_id=$1 AND status='ASSIGNED'`, candidateID).Scan(&total)
	rows, err := r.pool.Query(ctx,
		`SELECT id, exam_id, candidate_id, status, assigned_by, assigned_at, revoked_at
		 FROM exam_assignments WHERE candidate_id=$1 AND status='ASSIGNED' ORDER BY assigned_at DESC LIMIT $2 OFFSET $3`,
		candidateID, p.Limit(), p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.ExamAssignment
	for rows.Next() {
		var a models.ExamAssignment
		if err := rows.Scan(&a.ID, &a.ExamID, &a.CandidateID, &a.Status, &a.AssignedBy, &a.AssignedAt, &a.RevokedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (r *assignmentRepo) Create(ctx context.Context, a *models.ExamAssignment) error {
	if a.ID == uuid.Nil {
		a.ID = domain.NewID()
	}
	a.AssignedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO exam_assignments (id, exam_id, candidate_id, status, assigned_by, assigned_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		a.ID, a.ExamID, a.CandidateID, a.Status, a.AssignedBy, a.AssignedAt)
	return err
}

func (r *assignmentRepo) UpdateStatus(ctx context.Context, id domain.ID, status domain.AssignmentStatus, revokedAt interface{}) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE exam_assignments SET status=$2, revoked_at=$3 WHERE id=$1`, id, status, revokedAt)
	return err
}

// ---------------------------------------------------------------------------
// Attempt repository
// ---------------------------------------------------------------------------

type attemptRepo struct{ pool *pgxpool.Pool }

func (r *attemptRepo) FindByID(ctx context.Context, id domain.ID) (*models.ExamAttempt, error) {
	return findAttemptByID(ctx, r.pool, id, false)
}

func (r *attemptRepo) FindByIDForUpdate(ctx context.Context, id domain.ID) (*models.ExamAttempt, error) {
	return findAttemptByID(ctx, r.pool, id, true)
}

type attemptRepoTx struct {
	tx DBTX
}

func NewAttemptRepoTx(tx DBTX) repository.AttemptRepository {
	return &attemptRepoTx{tx: tx}
}

func (r *attemptRepoTx) FindByID(ctx context.Context, id domain.ID) (*models.ExamAttempt, error) {
	return findAttemptByID(ctx, r.tx, id, false)
}

func (r *attemptRepoTx) FindByIDForUpdate(ctx context.Context, id domain.ID) (*models.ExamAttempt, error) {
	return findAttemptByID(ctx, r.tx, id, true)
}

func (r *attemptRepoTx) FindByExamAndCandidate(ctx context.Context, examID, candidateID domain.ID) ([]models.ExamAttempt, error) {
	return findAttemptsByExamAndCandidate(ctx, r.tx, examID, candidateID)
}

func (r *attemptRepoTx) CountByAssignment(ctx context.Context, assignmentID domain.ID) (int, error) {
	var n int
	err := r.tx.QueryRow(ctx, `SELECT COUNT(*) FROM exam_attempts WHERE assignment_id=$1`, assignmentID).Scan(&n)
	return n, err
}

func (r *attemptRepoTx) List(ctx context.Context, p domain.Pagination) ([]models.ExamAttempt, int, error) {
	return listAttempts(ctx, r.tx, p)
}

func (r *attemptRepoTx) Create(ctx context.Context, a *models.ExamAttempt) error {
	return createAttempt(ctx, r.tx, a)
}

func (r *attemptRepoTx) Update(ctx context.Context, a *models.ExamAttempt) error {
	return updateAttempt(ctx, r.tx, a)
}

func findAttemptByID(ctx context.Context, db DBTX, id domain.ID, forUpdate bool) (*models.ExamAttempt, error) {
	a := &models.ExamAttempt{}
	query := `SELECT id, exam_id, candidate_id, assignment_id, status,
		        started_at, expires_at, submitted_at, finalized_at,
		        session_epoch, current_question_id, version, created_at, updated_at
		 FROM exam_attempts WHERE id=$1`
	if forUpdate {
		query += " FOR UPDATE"
	}
	err := db.QueryRow(ctx, query, id).
		Scan(&a.ID, &a.ExamID, &a.CandidateID, &a.AssignmentID, &a.Status,
			&a.StartedAt, &a.ExpiresAt, &a.SubmittedAt, &a.FinalizedAt,
			&a.SessionEpoch, &a.CurrentQuestionID, &a.Version, &a.CreatedAt, &a.UpdatedAt)
	return a, notFound(err)
}

func findAttemptsByExamAndCandidate(ctx context.Context, db DBTX, examID, candidateID domain.ID) ([]models.ExamAttempt, error) {
	rows, err := db.Query(ctx,
		`SELECT id, exam_id, candidate_id, assignment_id, status,
		        started_at, expires_at, submitted_at, finalized_at,
		        session_epoch, current_question_id, version, created_at, updated_at
		 FROM exam_attempts WHERE exam_id=$1 AND candidate_id=$2 ORDER BY created_at DESC`,
		examID, candidateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ExamAttempt
	for rows.Next() {
		var a models.ExamAttempt
		if err := rows.Scan(&a.ID, &a.ExamID, &a.CandidateID, &a.AssignmentID, &a.Status,
			&a.StartedAt, &a.ExpiresAt, &a.SubmittedAt, &a.FinalizedAt,
			&a.SessionEpoch, &a.CurrentQuestionID, &a.Version, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func listAttempts(ctx context.Context, db DBTX, p domain.Pagination) ([]models.ExamAttempt, int, error) {
	var total int
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM exam_attempts`).Scan(&total)
	rows, err := db.Query(ctx,
		`SELECT id, exam_id, candidate_id, assignment_id, status,
		        started_at, expires_at, submitted_at, finalized_at,
		        session_epoch, current_question_id, version, created_at, updated_at
		 FROM exam_attempts ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		p.Limit(), p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.ExamAttempt
	for rows.Next() {
		var a models.ExamAttempt
		if err := rows.Scan(&a.ID, &a.ExamID, &a.CandidateID, &a.AssignmentID, &a.Status,
			&a.StartedAt, &a.ExpiresAt, &a.SubmittedAt, &a.FinalizedAt,
			&a.SessionEpoch, &a.CurrentQuestionID, &a.Version, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func createAttempt(ctx context.Context, db DBTX, a *models.ExamAttempt) error {
	if a.ID == uuid.Nil {
		a.ID = domain.NewID()
	}
	now := time.Now().UTC()
	a.CreatedAt, a.UpdatedAt = now, now
	_, err := db.Exec(ctx,
		`INSERT INTO exam_attempts (id, exam_id, candidate_id, assignment_id, status,
		                            started_at, expires_at, submitted_at, finalized_at,
		                            session_epoch, current_question_id, version, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		a.ID, a.ExamID, a.CandidateID, a.AssignmentID, a.Status,
		a.StartedAt, a.ExpiresAt, a.SubmittedAt, a.FinalizedAt,
		a.SessionEpoch, a.CurrentQuestionID, a.Version, a.CreatedAt, a.UpdatedAt)
	return err
}

func updateAttempt(ctx context.Context, db DBTX, a *models.ExamAttempt) error {
	a.UpdatedAt = time.Now().UTC()
	a.Version++
	_, err := db.Exec(ctx,
		`UPDATE exam_attempts SET status=$2, started_at=$3, expires_at=$4, submitted_at=$5,
		                          finalized_at=$6, session_epoch=$7, current_question_id=$8,
		                          version=$9, updated_at=$10
		 WHERE id=$1`,
		a.ID, a.Status, a.StartedAt, a.ExpiresAt, a.SubmittedAt,
		a.FinalizedAt, a.SessionEpoch, a.CurrentQuestionID, a.Version, a.UpdatedAt)
	return err
}

func (r *attemptRepo) FindByExamAndCandidate(ctx context.Context, examID, candidateID domain.ID) ([]models.ExamAttempt, error) {
	return findAttemptsByExamAndCandidate(ctx, r.pool, examID, candidateID)
}

func (r *attemptRepo) CountByAssignment(ctx context.Context, assignmentID domain.ID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM exam_attempts WHERE assignment_id=$1`, assignmentID).Scan(&n)
	return n, err
}

func (r *attemptRepo) List(ctx context.Context, p domain.Pagination) ([]models.ExamAttempt, int, error) {
	return listAttempts(ctx, r.pool, p)
}

func (r *attemptRepo) Create(ctx context.Context, a *models.ExamAttempt) error {
	return createAttempt(ctx, r.pool, a)
}

func (r *attemptRepo) Update(ctx context.Context, a *models.ExamAttempt) error {
	return updateAttempt(ctx, r.pool, a)
}

// ---------------------------------------------------------------------------
// Session repository
// ---------------------------------------------------------------------------

type sessionRepo struct{ pool *pgxpool.Pool }

type sessionRepoTx struct {
	tx DBTX
}

func NewSessionRepoTx(tx DBTX) repository.SessionRepository {
	return &sessionRepoTx{tx: tx}
}

func (r *sessionRepo) FindByID(ctx context.Context, id domain.ID) (*models.AssessmentSession, error) {
	return findSessionByID(ctx, r.pool, id)
}

func (r *sessionRepo) FindActiveByAttempt(ctx context.Context, attemptID domain.ID) (*models.AssessmentSession, error) {
	return findActiveSessionByAttempt(ctx, r.pool, attemptID)
}

func (r *sessionRepo) Create(ctx context.Context, s *models.AssessmentSession) error {
	return createSession(ctx, r.pool, s)
}

func (r *sessionRepo) Update(ctx context.Context, s *models.AssessmentSession) error {
	return updateSession(ctx, r.pool, s)
}

func (r *sessionRepoTx) FindByID(ctx context.Context, id domain.ID) (*models.AssessmentSession, error) {
	return findSessionByID(ctx, r.tx, id)
}

func (r *sessionRepoTx) FindActiveByAttempt(ctx context.Context, attemptID domain.ID) (*models.AssessmentSession, error) {
	return findActiveSessionByAttempt(ctx, r.tx, attemptID)
}

func (r *sessionRepoTx) Create(ctx context.Context, s *models.AssessmentSession) error {
	return createSession(ctx, r.tx, s)
}

func (r *sessionRepoTx) Update(ctx context.Context, s *models.AssessmentSession) error {
	return updateSession(ctx, r.tx, s)
}

func findSessionByID(ctx context.Context, db DBTX, id domain.ID) (*models.AssessmentSession, error) {
	s := &models.AssessmentSession{}
	err := db.QueryRow(ctx,
		`SELECT id, attempt_id, client_instance_id, session_epoch, status,
		        connected_at, disconnected_at, terminated_at, last_heartbeat_at,
		        last_client_sequence, last_server_sequence, created_at, updated_at
		 FROM assessment_sessions WHERE id=$1`, id).
		Scan(&s.ID, &s.AttemptID, &s.ClientInstanceID, &s.SessionEpoch, &s.Status,
			&s.ConnectedAt, &s.DisconnectedAt, &s.TerminatedAt, &s.LastHeartbeatAt,
			&s.LastClientSequence, &s.LastServerSequence, &s.CreatedAt, &s.UpdatedAt)
	return s, notFound(err)
}

func findActiveSessionByAttempt(ctx context.Context, db DBTX, attemptID domain.ID) (*models.AssessmentSession, error) {
	s := &models.AssessmentSession{}
	err := db.QueryRow(ctx,
		`SELECT id, attempt_id, client_instance_id, session_epoch, status,
		        connected_at, disconnected_at, terminated_at, last_heartbeat_at,
		        last_client_sequence, last_server_sequence, created_at, updated_at
		 FROM assessment_sessions
		 WHERE attempt_id=$1 AND status NOT IN ('TERMINATED')
		 ORDER BY session_epoch DESC LIMIT 1`, attemptID).
		Scan(&s.ID, &s.AttemptID, &s.ClientInstanceID, &s.SessionEpoch, &s.Status,
			&s.ConnectedAt, &s.DisconnectedAt, &s.TerminatedAt, &s.LastHeartbeatAt,
			&s.LastClientSequence, &s.LastServerSequence, &s.CreatedAt, &s.UpdatedAt)
	return s, notFound(err)
}

func createSession(ctx context.Context, db DBTX, s *models.AssessmentSession) error {
	if s.ID == uuid.Nil {
		s.ID = domain.NewID()
	}
	now := time.Now().UTC()
	s.CreatedAt, s.UpdatedAt = now, now
	_, err := db.Exec(ctx,
		`INSERT INTO assessment_sessions (id, attempt_id, client_instance_id, session_epoch, status,
		                                  connected_at, last_client_sequence, last_server_sequence, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		s.ID, s.AttemptID, s.ClientInstanceID, s.SessionEpoch, s.Status,
		s.ConnectedAt, s.LastClientSequence, s.LastServerSequence, s.CreatedAt, s.UpdatedAt)
	return err
}

func updateSession(ctx context.Context, db DBTX, s *models.AssessmentSession) error {
	s.UpdatedAt = time.Now().UTC()
	_, err := db.Exec(ctx,
		`UPDATE assessment_sessions SET status=$2, connected_at=$3, disconnected_at=$4,
		                                terminated_at=$5, last_heartbeat_at=$6,
		                                last_client_sequence=$7, last_server_sequence=$8, updated_at=$9
		 WHERE id=$1`,
		s.ID, s.Status, s.ConnectedAt, s.DisconnectedAt, s.TerminatedAt,
		s.LastHeartbeatAt, s.LastClientSequence, s.LastServerSequence, s.UpdatedAt)
	return err
}

// ---------------------------------------------------------------------------
// Response repository
// ---------------------------------------------------------------------------

type responseRepo struct{ pool *pgxpool.Pool }

type responseRepoTx struct {
	tx DBTX
}

func NewResponseRepoTx(tx DBTX) repository.ResponseRepository {
	return &responseRepoTx{tx: tx}
}

func (r *responseRepo) FindByAttemptAndQuestion(ctx context.Context, attemptID, questionVersionID domain.ID) (*models.CandidateResponse, error) {
	return findResponseByAttemptAndQuestion(ctx, r.pool, attemptID, questionVersionID)
}

func (r *responseRepo) ListByAttempt(ctx context.Context, attemptID domain.ID) ([]models.CandidateResponse, error) {
	return listResponsesByAttempt(ctx, r.pool, attemptID)
}

func (r *responseRepo) Upsert(ctx context.Context, cr *models.CandidateResponse) error {
	return upsertResponse(ctx, r.pool, cr)
}

func (r *responseRepo) UpdateStatus(ctx context.Context, attemptID domain.ID, status domain.ResponseStatus) error {
	return updateResponseStatus(ctx, r.pool, attemptID, status)
}

func (r *responseRepoTx) FindByAttemptAndQuestion(ctx context.Context, attemptID, questionVersionID domain.ID) (*models.CandidateResponse, error) {
	return findResponseByAttemptAndQuestion(ctx, r.tx, attemptID, questionVersionID)
}

func (r *responseRepoTx) ListByAttempt(ctx context.Context, attemptID domain.ID) ([]models.CandidateResponse, error) {
	return listResponsesByAttempt(ctx, r.tx, attemptID)
}

func (r *responseRepoTx) Upsert(ctx context.Context, cr *models.CandidateResponse) error {
	return upsertResponse(ctx, r.tx, cr)
}

func (r *responseRepoTx) UpdateStatus(ctx context.Context, attemptID domain.ID, status domain.ResponseStatus) error {
	return updateResponseStatus(ctx, r.tx, attemptID, status)
}

func findResponseByAttemptAndQuestion(ctx context.Context, db DBTX, attemptID, questionVersionID domain.ID) (*models.CandidateResponse, error) {
	cr := &models.CandidateResponse{}
	err := db.QueryRow(ctx,
		`SELECT id, attempt_id, question_version_id, current_value, status, version, updated_at
		 FROM candidate_responses WHERE attempt_id=$1 AND question_version_id=$2`,
		attemptID, questionVersionID).
		Scan(&cr.ID, &cr.AttemptID, &cr.QuestionVersionID, &cr.CurrentValue, &cr.Status, &cr.Version, &cr.UpdatedAt)
	return cr, notFound(err)
}

func listResponsesByAttempt(ctx context.Context, db DBTX, attemptID domain.ID) ([]models.CandidateResponse, error) {
	rows, err := db.Query(ctx,
		`SELECT id, attempt_id, question_version_id, current_value, status, version, updated_at
		 FROM candidate_responses WHERE attempt_id=$1`, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.CandidateResponse
	for rows.Next() {
		var cr models.CandidateResponse
		if err := rows.Scan(&cr.ID, &cr.AttemptID, &cr.QuestionVersionID, &cr.CurrentValue, &cr.Status, &cr.Version, &cr.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, cr)
	}
	return out, rows.Err()
}

func upsertResponse(ctx context.Context, db DBTX, cr *models.CandidateResponse) error {
	if cr.ID == uuid.Nil {
		cr.ID = domain.NewID()
	}
	cr.UpdatedAt = time.Now().UTC()
	_, err := db.Exec(ctx,
		`INSERT INTO candidate_responses (id, attempt_id, question_version_id, current_value, status, version, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (attempt_id, question_version_id) DO UPDATE
		 SET current_value = EXCLUDED.current_value, status = EXCLUDED.status,
		     version = candidate_responses.version + 1, updated_at = EXCLUDED.updated_at`,
		cr.ID, cr.AttemptID, cr.QuestionVersionID, cr.CurrentValue, cr.Status, cr.Version, cr.UpdatedAt)
	return err
}

func updateResponseStatus(ctx context.Context, db DBTX, attemptID domain.ID, status domain.ResponseStatus) error {
	_, err := db.Exec(ctx,
		`UPDATE candidate_responses SET status=$2, updated_at=NOW() WHERE attempt_id=$1`, attemptID, status)
	return err
}

// ---------------------------------------------------------------------------
// Submission repository
// ---------------------------------------------------------------------------

type submissionRepo struct{ pool *pgxpool.Pool }

func (r *submissionRepo) FindByAttempt(ctx context.Context, attemptID domain.ID) (*models.Submission, error) {
	s := &models.Submission{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, attempt_id, status, requested_at, submitted_at, finalized_at, score, max_score, created_at, updated_at
		 FROM submissions WHERE attempt_id=$1`, attemptID).
		Scan(&s.ID, &s.AttemptID, &s.Status, &s.RequestedAt, &s.SubmittedAt,
			&s.FinalizedAt, &s.Score, &s.MaxScore, &s.CreatedAt, &s.UpdatedAt)
	return s, notFound(err)
}

func (r *submissionRepo) Create(ctx context.Context, s *models.Submission) error {
	if s.ID == uuid.Nil {
		s.ID = domain.NewID()
	}
	now := time.Now().UTC()
	s.CreatedAt, s.UpdatedAt = now, now
	_, err := r.pool.Exec(ctx,
		`INSERT INTO submissions (id, attempt_id, status, requested_at, submitted_at, finalized_at, score, max_score, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		s.ID, s.AttemptID, s.Status, s.RequestedAt, s.SubmittedAt,
		s.FinalizedAt, s.Score, s.MaxScore, s.CreatedAt, s.UpdatedAt)
	return err
}

func (r *submissionRepo) Update(ctx context.Context, s *models.Submission) error {
	s.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE submissions SET status=$2, submitted_at=$3, finalized_at=$4, score=$5, max_score=$6, updated_at=$7
		 WHERE id=$1`,
		s.ID, s.Status, s.SubmittedAt, s.FinalizedAt, s.Score, s.MaxScore, s.UpdatedAt)
	return err
}

// ---------------------------------------------------------------------------
// DBTX interface and DB accessors
// ---------------------------------------------------------------------------

type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (d *DB) Pool() *pgxpool.Pool {
	return d.pool
}

// RepositoriesTx builds repository instances operating within a given transaction or DBTX.
func RepositoriesTx(tx DBTX) repository.Repositories {
	return repository.Repositories{
		Attempts:        NewAttemptRepoTx(tx),
		Sessions:        NewSessionRepoTx(tx),
		Responses:       NewResponseRepoTx(tx),
		CandidateEvents: NewCandidateEventRepoTx(tx),
		OutboxEvents:    NewOutboxRepoTx(tx),
	}
}

// ---------------------------------------------------------------------------
// CandidateEvent repository
// ---------------------------------------------------------------------------

type candidateEventRepo struct {
	pool *pgxpool.Pool
}

func NewCandidateEventRepo(pool *pgxpool.Pool) repository.CandidateEventRepository {
	return &candidateEventRepo{pool: pool}
}

// CandidateEventRepoTx allows running candidateEventRepo against a transaction or connection pool.
type candidateEventRepoTx struct {
	tx DBTX
}

func NewCandidateEventRepoTx(tx DBTX) repository.CandidateEventRepository {
	return &candidateEventRepoTx{tx: tx}
}

func (r *candidateEventRepo) Insert(ctx context.Context, event *models.CandidateEvent) error {
	return insertCandidateEvent(ctx, r.pool, event)
}

func (r *candidateEventRepo) FindByEventID(ctx context.Context, eventID domain.ID) (*models.CandidateEvent, error) {
	return findCandidateEventByEventID(ctx, r.pool, eventID)
}

func (r *candidateEventRepo) FindByAttemptAndSequence(ctx context.Context, attemptID domain.ID, seq int64) (*models.CandidateEvent, error) {
	return findCandidateEventByAttemptAndSequence(ctx, r.pool, attemptID, seq)
}

func (r *candidateEventRepo) GetMaxSequence(ctx context.Context, attemptID domain.ID) (int64, error) {
	return getCandidateEventMaxSequence(ctx, r.pool, attemptID)
}

func (r *candidateEventRepo) FindLatestByAttempt(ctx context.Context, attemptID domain.ID) (*models.CandidateEvent, error) {
	return findCandidateEventLatestByAttempt(ctx, r.pool, attemptID)
}

func (r *candidateEventRepo) ListByAttemptRange(ctx context.Context, attemptID domain.ID, fromSeq, toSeq int64) ([]models.CandidateEvent, error) {
	return listCandidateEventsByRange(ctx, r.pool, attemptID, fromSeq, toSeq)
}

func (r *candidateEventRepo) ListByAttempt(ctx context.Context, attemptID domain.ID) ([]models.CandidateEvent, error) {
	return listCandidateEventsByAttempt(ctx, r.pool, attemptID)
}

func (r *candidateEventRepoTx) Insert(ctx context.Context, event *models.CandidateEvent) error {
	return insertCandidateEvent(ctx, r.tx, event)
}

func (r *candidateEventRepoTx) FindByEventID(ctx context.Context, eventID domain.ID) (*models.CandidateEvent, error) {
	return findCandidateEventByEventID(ctx, r.tx, eventID)
}

func (r *candidateEventRepoTx) FindByAttemptAndSequence(ctx context.Context, attemptID domain.ID, seq int64) (*models.CandidateEvent, error) {
	return findCandidateEventByAttemptAndSequence(ctx, r.tx, attemptID, seq)
}

func (r *candidateEventRepoTx) GetMaxSequence(ctx context.Context, attemptID domain.ID) (int64, error) {
	return getCandidateEventMaxSequence(ctx, r.tx, attemptID)
}

func (r *candidateEventRepoTx) FindLatestByAttempt(ctx context.Context, attemptID domain.ID) (*models.CandidateEvent, error) {
	return findCandidateEventLatestByAttempt(ctx, r.tx, attemptID)
}

func (r *candidateEventRepoTx) ListByAttemptRange(ctx context.Context, attemptID domain.ID, fromSeq, toSeq int64) ([]models.CandidateEvent, error) {
	return listCandidateEventsByRange(ctx, r.tx, attemptID, fromSeq, toSeq)
}

func (r *candidateEventRepoTx) ListByAttempt(ctx context.Context, attemptID domain.ID) ([]models.CandidateEvent, error) {
	return listCandidateEventsByAttempt(ctx, r.tx, attemptID)
}

func insertCandidateEvent(ctx context.Context, db DBTX, event *models.CandidateEvent) error {
	if event.ID == uuid.Nil {
		event.ID = domain.NewID()
	}
	if event.SchemaVersion == "" {
		event.SchemaVersion = "1.0"
	}
	if event.ProcessingStatus == "" {
		event.ProcessingStatus = domain.EventProcessingStatusCommitted
	}
	if event.ServerReceivedAt.IsZero() {
		event.ServerReceivedAt = time.Now().UTC()
	}
	if event.Payload == nil {
		event.Payload = map[string]any{}
	}

	_, err := db.Exec(ctx,
		`INSERT INTO candidate_events (
			id, event_id, attempt_id, session_id, session_epoch,
			sequence_number, event_type, schema_version, client_instance_id,
			payload, client_timestamp, server_received_at, prev_hash,
			event_hash, processing_status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		event.ID, event.EventID, event.AttemptID, event.SessionID, event.SessionEpoch,
		event.SequenceNumber, string(event.EventType), event.SchemaVersion, event.ClientInstanceID,
		event.Payload, event.ClientTimestamp, event.ServerReceivedAt, event.PrevHash,
		event.EventHash, string(event.ProcessingStatus))
	return err
}

func findCandidateEventByEventID(ctx context.Context, db DBTX, eventID domain.ID) (*models.CandidateEvent, error) {
	e := &models.CandidateEvent{}
	err := db.QueryRow(ctx,
		`SELECT id, event_id, attempt_id, session_id, session_epoch,
		        sequence_number, event_type, schema_version, client_instance_id,
		        payload, client_timestamp, server_received_at, prev_hash,
		        event_hash, processing_status
		 FROM candidate_events WHERE event_id=$1`, eventID).
		Scan(&e.ID, &e.EventID, &e.AttemptID, &e.SessionID, &e.SessionEpoch,
			&e.SequenceNumber, &e.EventType, &e.SchemaVersion, &e.ClientInstanceID,
			&e.Payload, &e.ClientTimestamp, &e.ServerReceivedAt, &e.PrevHash,
			&e.EventHash, &e.ProcessingStatus)
	return e, notFound(err)
}

func findCandidateEventByAttemptAndSequence(ctx context.Context, db DBTX, attemptID domain.ID, seq int64) (*models.CandidateEvent, error) {
	e := &models.CandidateEvent{}
	err := db.QueryRow(ctx,
		`SELECT id, event_id, attempt_id, session_id, session_epoch,
		        sequence_number, event_type, schema_version, client_instance_id,
		        payload, client_timestamp, server_received_at, prev_hash,
		        event_hash, processing_status
		 FROM candidate_events WHERE attempt_id=$1 AND sequence_number=$2`, attemptID, seq).
		Scan(&e.ID, &e.EventID, &e.AttemptID, &e.SessionID, &e.SessionEpoch,
			&e.SequenceNumber, &e.EventType, &e.SchemaVersion, &e.ClientInstanceID,
			&e.Payload, &e.ClientTimestamp, &e.ServerReceivedAt, &e.PrevHash,
			&e.EventHash, &e.ProcessingStatus)
	return e, notFound(err)
}

func getCandidateEventMaxSequence(ctx context.Context, db DBTX, attemptID domain.ID) (int64, error) {
	var maxSeq *int64
	err := db.QueryRow(ctx,
		`SELECT MAX(sequence_number) FROM candidate_events WHERE attempt_id=$1`, attemptID).
		Scan(&maxSeq)
	if err != nil {
		return 0, err
	}
	if maxSeq == nil {
		return 0, nil
	}
	return *maxSeq, nil
}

func findCandidateEventLatestByAttempt(ctx context.Context, db DBTX, attemptID domain.ID) (*models.CandidateEvent, error) {
	e := &models.CandidateEvent{}
	err := db.QueryRow(ctx,
		`SELECT id, event_id, attempt_id, session_id, session_epoch,
		        sequence_number, event_type, schema_version, client_instance_id,
		        payload, client_timestamp, server_received_at, prev_hash,
		        event_hash, processing_status
		 FROM candidate_events
		 WHERE attempt_id=$1
		 ORDER BY sequence_number DESC LIMIT 1`, attemptID).
		Scan(&e.ID, &e.EventID, &e.AttemptID, &e.SessionID, &e.SessionEpoch,
			&e.SequenceNumber, &e.EventType, &e.SchemaVersion, &e.ClientInstanceID,
			&e.Payload, &e.ClientTimestamp, &e.ServerReceivedAt, &e.PrevHash,
			&e.EventHash, &e.ProcessingStatus)
	return e, notFound(err)
}

func listCandidateEventsByRange(ctx context.Context, db DBTX, attemptID domain.ID, fromSeq, toSeq int64) ([]models.CandidateEvent, error) {
	rows, err := db.Query(ctx,
		`SELECT id, event_id, attempt_id, session_id, session_epoch,
		        sequence_number, event_type, schema_version, client_instance_id,
		        payload, client_timestamp, server_received_at, prev_hash,
		        event_hash, processing_status
		 FROM candidate_events
		 WHERE attempt_id=$1 AND sequence_number >= $2 AND sequence_number <= $3
		 ORDER BY sequence_number ASC`, attemptID, fromSeq, toSeq)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.CandidateEvent
	for rows.Next() {
		var e models.CandidateEvent
		if err := rows.Scan(&e.ID, &e.EventID, &e.AttemptID, &e.SessionID, &e.SessionEpoch,
			&e.SequenceNumber, &e.EventType, &e.SchemaVersion, &e.ClientInstanceID,
			&e.Payload, &e.ClientTimestamp, &e.ServerReceivedAt, &e.PrevHash,
			&e.EventHash, &e.ProcessingStatus); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func listCandidateEventsByAttempt(ctx context.Context, db DBTX, attemptID domain.ID) ([]models.CandidateEvent, error) {
	rows, err := db.Query(ctx,
		`SELECT id, event_id, attempt_id, session_id, session_epoch,
		        sequence_number, event_type, schema_version, client_instance_id,
		        payload, client_timestamp, server_received_at, prev_hash,
		        event_hash, processing_status
		 FROM candidate_events
		 WHERE attempt_id=$1
		 ORDER BY sequence_number ASC`, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.CandidateEvent
	for rows.Next() {
		var e models.CandidateEvent
		if err := rows.Scan(&e.ID, &e.EventID, &e.AttemptID, &e.SessionID, &e.SessionEpoch,
			&e.SequenceNumber, &e.EventType, &e.SchemaVersion, &e.ClientInstanceID,
			&e.Payload, &e.ClientTimestamp, &e.ServerReceivedAt, &e.PrevHash,
			&e.EventHash, &e.ProcessingStatus); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Outbox repository
// ---------------------------------------------------------------------------

type outboxRepo struct {
	pool *pgxpool.Pool
}

func NewOutboxRepo(pool *pgxpool.Pool) repository.OutboxRepository {
	return &outboxRepo{pool: pool}
}

type outboxRepoTx struct {
	tx DBTX
}

func NewOutboxRepoTx(tx DBTX) repository.OutboxRepository {
	return &outboxRepoTx{tx: tx}
}

func (r *outboxRepo) Create(ctx context.Context, event *models.OutboxEvent) error {
	return createOutboxEvent(ctx, r.pool, event)
}

func (r *outboxRepo) ListPending(ctx context.Context, limit int) ([]models.OutboxEvent, error) {
	return listPendingOutboxEvents(ctx, r.pool, limit)
}

func (r *outboxRepo) MarkPublished(ctx context.Context, id domain.ID) error {
	return markOutboxEventPublished(ctx, r.pool, id)
}

func (r *outboxRepoTx) Create(ctx context.Context, event *models.OutboxEvent) error {
	return createOutboxEvent(ctx, r.tx, event)
}

func (r *outboxRepoTx) ListPending(ctx context.Context, limit int) ([]models.OutboxEvent, error) {
	return listPendingOutboxEvents(ctx, r.tx, limit)
}

func (r *outboxRepoTx) MarkPublished(ctx context.Context, id domain.ID) error {
	return markOutboxEventPublished(ctx, r.tx, id)
}

func createOutboxEvent(ctx context.Context, db DBTX, event *models.OutboxEvent) error {
	if event.ID == uuid.Nil {
		event.ID = domain.NewID()
	}
	if event.Status == "" {
		event.Status = domain.OutboxStatusPending
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	_, err := db.Exec(ctx,
		`INSERT INTO outbox_events (id, event_type, stream_name, subject, payload, status, created_at, published_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		event.ID, event.EventType, event.StreamName, event.Subject, event.Payload,
		string(event.Status), event.CreatedAt, event.PublishedAt)
	return err
}

func listPendingOutboxEvents(ctx context.Context, db DBTX, limit int) ([]models.OutboxEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Query(ctx,
		`SELECT id, event_type, stream_name, subject, payload, status, created_at, published_at
		 FROM outbox_events
		 WHERE status = 'PENDING'
		 ORDER BY created_at ASC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.OutboxEvent
	for rows.Next() {
		var e models.OutboxEvent
		if err := rows.Scan(&e.ID, &e.EventType, &e.StreamName, &e.Subject, &e.Payload,
			&e.Status, &e.CreatedAt, &e.PublishedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func markOutboxEventPublished(ctx context.Context, db DBTX, id domain.ID) error {
	now := time.Now().UTC()
	_, err := db.Exec(ctx,
		`UPDATE outbox_events SET status='PUBLISHED', published_at=$2 WHERE id=$1`, id, now)
	return err
}

