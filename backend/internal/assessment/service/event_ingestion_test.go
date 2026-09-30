package service_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentinelexam/backend/internal/assessment/domain"
	"github.com/sentinelexam/backend/internal/assessment/models"
	repopg "github.com/sentinelexam/backend/internal/assessment/repository/postgres"
	"github.com/sentinelexam/backend/internal/assessment/service"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	host := os.Getenv("DATABASE_HOST")
	if host == "" {
		host = "postgres"
	}
	port := os.Getenv("DATABASE_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DATABASE_USER")
	if user == "" {
		user = "sentinel"
	}
	password := os.Getenv("DATABASE_PASSWORD")
	if password == "" {
		password = "sentinel_dev_password"
	}
	dbname := os.Getenv("DATABASE_NAME")
	if dbname == "" {
		dbname = "sentinelexam"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping test: cannot connect to postgres at %s: %v", dsn, err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping test: postgres ping failed: %v", err)
	}
	return pool
}

type testFixture struct {
	User      models.User
	Candidate models.Candidate
	Exam      models.Exam
	QuestionV models.QuestionVersion
	Attempt   models.ExamAttempt
	Session   models.AssessmentSession
	Claims    service.AuthClaims
}

func setupTestFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (*testFixture, *service.EventIngestionService) {
	t.Helper()
	db := repopg.New(pool)
	repos := db.Repositories()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ingestSvc := service.NewEventIngestionService(pool, repos, logger)

	// User
	userID := domain.NewID()
	sub := domain.NewID()
	user := models.User{
		ID:          userID,
		KeycloakSub: sub,
		Email:       fmt.Sprintf("cand-%s@example.com", uuid.New().String()[:8]),
		DisplayName: "Test Candidate",
		Roles:       []string{"candidate"},
	}
	if err := repos.Users.Upsert(ctx, &user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Candidate
	cand := models.Candidate{
		ID:       domain.NewID(),
		UserID:   userID,
		Status:   domain.CandidateStatusActive,
		FullName: "Test Candidate",
		Metadata: map[string]any{},
	}
	if err := repos.Candidates.Create(ctx, &cand); err != nil {
		t.Fatalf("failed to create candidate: %v", err)
	}

	// Question Bank & Question & Version
	bank := models.QuestionBank{
		ID:     domain.NewID(),
		Name:   "Test Bank",
		Status: domain.QuestionBankStatusActive,
	}
	if err := repos.QuestionBanks.Create(ctx, &bank); err != nil {
		t.Fatalf("failed to create qb: %v", err)
	}
	q := models.Question{
		ID:             domain.NewID(),
		QuestionBankID: bank.ID,
		Status:         domain.QuestionStatusActive,
	}
	if err := repos.Questions.Create(ctx, &q); err != nil {
		t.Fatalf("failed to create question: %v", err)
	}
	qv := models.QuestionVersion{
		ID:         domain.NewID(),
		QuestionID: q.ID,
		Version:    1,
		Type:       domain.QuestionTypeSingleChoice,
		Content: map[string]any{
			"prompt":  "What is 2+2?",
			"options": []string{"3", "4", "5"},
		},
		CorrectAnswer: map[string]any{
			"selected": "4",
		},
		Marks:  1.0,
		Status: domain.QuestionStatusActive,
	}
	if err := repos.QuestionVersions.Create(ctx, &qv); err != nil {
		t.Fatalf("failed to create question version: %v", err)
	}

	// Exam
	exam := models.Exam{
		ID:              domain.NewID(),
		Title:           "Resilience Ingestion Test Exam",
		Status:          domain.ExamStatusLive,
		DurationSeconds: 3600,
		MaxAttempts:     1,
		Navigation:      map[string]any{"allow_backtracking": true},
		Scoring:         map[string]any{"mode": "STANDARD"},
		Version:         1,
	}
	if err := repos.Exams.Create(ctx, &exam); err != nil {
		t.Fatalf("failed to create exam: %v", err)
	}
	if err := repos.ExamQuestions.Add(ctx, &models.ExamQuestion{
		ID:                domain.NewID(),
		ExamID:            exam.ID,
		QuestionVersionID: qv.ID,
		Position:          1,
	}); err != nil {
		t.Fatalf("failed to add exam question: %v", err)
	}

	// Assignment
	assign := models.ExamAssignment{
		ID:          domain.NewID(),
		ExamID:      exam.ID,
		CandidateID: cand.ID,
		Status:      domain.AssignmentStatusAssigned,
		AssignedAt:  time.Now().UTC(),
	}
	if err := repos.Assignments.Create(ctx, &assign); err != nil {
		t.Fatalf("failed to create assignment: %v", err)
	}

	// Attempt
	now := time.Now().UTC()
	exp := now.Add(1 * time.Hour)
	attempt := models.ExamAttempt{
		ID:           domain.NewID(),
		ExamID:       exam.ID,
		CandidateID:  cand.ID,
		AssignmentID: assign.ID,
		Status:       domain.AttemptStatusActive,
		StartedAt:    &now,
		ExpiresAt:    &exp,
		SessionEpoch: 1,
		Version:      1,
	}
	if err := repos.Attempts.Create(ctx, &attempt); err != nil {
		t.Fatalf("failed to create attempt: %v", err)
	}

	// Session
	sess := models.AssessmentSession{
		ID:               domain.NewID(),
		AttemptID:        attempt.ID,
		ClientInstanceID: "tab-test-instance",
		SessionEpoch:     1,
		Status:           domain.SessionStatusConnected,
		ConnectedAt:      &now,
	}
	if err := repos.Sessions.Create(ctx, &sess); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	claims := service.AuthClaims{
		Sub:         sub.String(),
		KeycloakSub: sub,
		Email:       user.Email,
		Roles:       []string{"candidate"},
	}

	return &testFixture{
		User:      user,
		Candidate: cand,
		Exam:      exam,
		QuestionV: qv,
		Attempt:   attempt,
		Session:   sess,
		Claims:    claims,
	}, ingestSvc
}

func TestEventIngestion_ValidEventAndHashChaining(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	ctx := context.Background()
	fix, svc := setupTestFixture(t, ctx, pool)

	eventID1 := domain.NewID()
	t1 := time.Now().UTC().Add(-10 * time.Second)
	input := service.SyncEventsInput{
		AttemptID: fix.Attempt.ID,
		Claims:    fix.Claims,
		Events: []service.SyncEventItemInput{
			{
				EventID:          eventID1,
				SessionID:        &fix.Session.ID,
				SessionEpoch:     1,
				SequenceNumber:   1,
				EventType:        "candidate.option_selected",
				SchemaVersion:    "1.0",
				ClientInstanceID: fix.Session.ClientInstanceID,
				ClientTimestamp:  t1,
				Payload: map[string]any{
					"question_version_id": fix.QuestionV.ID.String(),
					"value":               map[string]any{"selected": "B"},
				},
			},
		},
	}

	res, err := svc.SyncEvents(ctx, input)
	if err != nil {
		t.Fatalf("SyncEvents failed: %v", err)
	}
	if res.AcceptedCount != 1 {
		t.Fatalf("expected accepted_count 1, got %d", res.AcceptedCount)
	}
	if res.LastCommittedSequence != 1 {
		t.Fatalf("expected last_committed_sequence 1, got %d", res.LastCommittedSequence)
	}
	if res.ServerTimestamp.IsZero() {
		t.Fatalf("server_timestamp is zero")
	}

	// 1. Verify candidate event in DB
	repos := repopg.New(pool).Repositories()
	evt1, err := repos.CandidateEvents.FindByEventID(ctx, eventID1)
	if err != nil {
		t.Fatalf("failed to find persisted candidate event: %v", err)
	}
	if evt1.SequenceNumber != 1 {
		t.Fatalf("expected sequence 1, got %d", evt1.SequenceNumber)
	}
	if evt1.EventType != domain.CandidateEventTypeOptionSelected {
		t.Fatalf("expected normalized event type %s, got %s", domain.CandidateEventTypeOptionSelected, evt1.EventType)
	}
	if evt1.PrevHash == nil || *evt1.PrevHash == "" {
		t.Fatalf("expected non-empty prev_hash on initial event")
	}
	if evt1.EventHash == nil || *evt1.EventHash == "" {
		t.Fatalf("expected non-empty event_hash on initial event")
	}

	// Verify initial prev_hash formula: H_0 = SHA256(AttemptID + StartTimestamp)
	startStr := fmt.Sprintf("%d", fix.Attempt.StartedAt.UnixMilli())
	h0Sum := sha256.Sum256([]byte(fix.Attempt.ID.String() + startStr))
	expectedH0 := hex.EncodeToString(h0Sum[:])
	if *evt1.PrevHash != expectedH0 {
		t.Fatalf("H_0 mismatch: got %s, want %s", *evt1.PrevHash, expectedH0)
	}

	// Verify H_1 formula: SHA256(PrevHash + Seq + Type + PayloadJSON + ClientTimestamp)
	payloadBytes, _ := json.Marshal(input.Events[0].Payload)
	rawChain1 := fmt.Sprintf("%s%d%s%s%s",
		expectedH0,
		1,
		string(domain.CandidateEventTypeOptionSelected),
		string(payloadBytes),
		t1.Format(time.RFC3339Nano),
	)
	h1Sum := sha256.Sum256([]byte(rawChain1))
	expectedH1 := hex.EncodeToString(h1Sum[:])
	if *evt1.EventHash != expectedH1 {
		t.Fatalf("H_1 mismatch: got %s, want %s", *evt1.EventHash, expectedH1)
	}

	// 2. Verify candidate response projection
	resp, err := repos.Responses.FindByAttemptAndQuestion(ctx, fix.Attempt.ID, fix.QuestionV.ID)
	if err != nil {
		t.Fatalf("failed to find projected response: %v", err)
	}
	if resp.Status != domain.ResponseStatusAnswered {
		t.Fatalf("expected response status ANSWERED, got %s", resp.Status)
	}

	// 3. Verify Outbox Event created
	outboxList, err := repos.OutboxEvents.ListPending(ctx, 1000)
	if err != nil {
		t.Fatalf("failed to query outbox: %v", err)
	}
	var foundOutbox bool
	for _, o := range outboxList {
		if payload, ok := o.Payload["data"].(map[string]any); ok {
			if payload["attempt_id"] == fix.Attempt.ID.String() && payload["sequence_number"] == float64(1) {
				foundOutbox = true
				break
			}
		}
	}
	if !foundOutbox {
		for _, o := range outboxList {
			fmt.Printf("DEBUG Outbox: ID=%v Sequence=%v PayloadData=%+v\n", o.ID, o.Payload["data"], o.Payload["data"])
		}
		t.Fatalf("expected outbox event for attempt %s sequence 1", fix.Attempt.ID)
	}

	// 4. Insert Second Event (Sequence 2) and verify hash continuity
	eventID2 := domain.NewID()
	t2 := time.Now().UTC()
	input2 := service.SyncEventsInput{
		AttemptID: fix.Attempt.ID,
		Claims:    fix.Claims,
		Events: []service.SyncEventItemInput{
			{
				EventID:          eventID2,
				SessionID:        &fix.Session.ID,
				SessionEpoch:     1,
				SequenceNumber:   2,
				EventType:        "candidate.question_navigated",
				SchemaVersion:    "1.0",
				ClientInstanceID: fix.Session.ClientInstanceID,
				ClientTimestamp:  t2,
				Payload:          map[string]any{"target_question": 2},
			},
		},
	}
	res2, err := svc.SyncEvents(ctx, input2)
	if err != nil {
		t.Fatalf("SyncEvents sequence 2 failed: %v", err)
	}
	if res2.AcceptedCount != 1 || res2.LastCommittedSequence != 2 {
		t.Fatalf("unexpected res2: %+v", res2)
	}

	evt2, err := repos.CandidateEvents.FindByEventID(ctx, eventID2)
	if err != nil {
		t.Fatalf("failed to find event 2: %v", err)
	}
	if *evt2.PrevHash != *evt1.EventHash {
		t.Fatalf("hash chain broken! evt2.PrevHash (%s) != evt1.EventHash (%s)", *evt2.PrevHash, *evt1.EventHash)
	}
}

func TestEventIngestion_DuplicateEventID_IdempotentSuccess(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	ctx := context.Background()
	fix, svc := setupTestFixture(t, ctx, pool)

	eventID := domain.NewID()
	evtInput := service.SyncEventItemInput{
		EventID:          eventID,
		SessionID:        &fix.Session.ID,
		SessionEpoch:     1,
		SequenceNumber:   1,
		EventType:        "candidate.option_selected",
		SchemaVersion:    "1.0",
		ClientInstanceID: fix.Session.ClientInstanceID,
		ClientTimestamp:  time.Now().UTC(),
		Payload:          map[string]any{"selected": "A"},
	}

	// First submission
	res1, err := svc.SyncEvents(ctx, service.SyncEventsInput{
		AttemptID: fix.Attempt.ID,
		Claims:    fix.Claims,
		Events:    []service.SyncEventItemInput{evtInput},
	})
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}
	if res1.AcceptedCount != 1 {
		t.Fatalf("expected accepted_count 1, got %d", res1.AcceptedCount)
	}

	// Duplicate retry with exact same event_id and sequence
	res2, err := svc.SyncEvents(ctx, service.SyncEventsInput{
		AttemptID: fix.Attempt.ID,
		Claims:    fix.Claims,
		Events:    []service.SyncEventItemInput{evtInput},
	})
	if err != nil {
		t.Fatalf("duplicate retry failed: %v", err)
	}
	// Accepted count for duplicate is 0 (idempotently skipped), but returns success and current committed sequence
	if res2.AcceptedCount != 0 {
		t.Fatalf("expected accepted_count 0 for duplicate, got %d", res2.AcceptedCount)
	}
	if res2.LastCommittedSequence != 1 {
		t.Fatalf("expected last_committed_sequence 1, got %d", res2.LastCommittedSequence)
	}
}

func TestEventIngestion_SameSequence_DifferentEventID_Conflict(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	ctx := context.Background()
	fix, svc := setupTestFixture(t, ctx, pool)

	// First event sequence 1
	_, err := svc.SyncEvents(ctx, service.SyncEventsInput{
		AttemptID: fix.Attempt.ID,
		Claims:    fix.Claims,
		Events: []service.SyncEventItemInput{
			{
				EventID:          domain.NewID(),
				SessionID:        &fix.Session.ID,
				SessionEpoch:     1,
				SequenceNumber:   1,
				EventType:        "candidate.option_selected",
				SchemaVersion:    "1.0",
				ClientInstanceID: fix.Session.ClientInstanceID,
				ClientTimestamp:  time.Now().UTC(),
				Payload:          map[string]any{"selected": "A"},
			},
		},
	})
	if err != nil {
		t.Fatalf("first event failed: %v", err)
	}

	// Different event_id with same sequence number 1 -> Conflict
	_, err = svc.SyncEvents(ctx, service.SyncEventsInput{
		AttemptID: fix.Attempt.ID,
		Claims:    fix.Claims,
		Events: []service.SyncEventItemInput{
			{
				EventID:          domain.NewID(), // New event ID
				SessionID:        &fix.Session.ID,
				SessionEpoch:     1,
				SequenceNumber:   1, // Same sequence 1
				EventType:        "candidate.option_selected",
				SchemaVersion:    "1.0",
				ClientInstanceID: fix.Session.ClientInstanceID,
				ClientTimestamp:  time.Now().UTC(),
				Payload:          map[string]any{"selected": "B"},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected conflict for same sequence with different event_id, got nil")
	}
}

func TestEventIngestion_SequenceGap_Conflict(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	ctx := context.Background()
	fix, svc := setupTestFixture(t, ctx, pool)

	// Attempt to insert sequence 5 when max sequence is 0 -> Gap conflict
	_, err := svc.SyncEvents(ctx, service.SyncEventsInput{
		AttemptID: fix.Attempt.ID,
		Claims:    fix.Claims,
		Events: []service.SyncEventItemInput{
			{
				EventID:          domain.NewID(),
				SessionID:        &fix.Session.ID,
				SessionEpoch:     1,
				SequenceNumber:   5, // Gap (expected 1)
				EventType:        "candidate.option_selected",
				SchemaVersion:    "1.0",
				ClientInstanceID: fix.Session.ClientInstanceID,
				ClientTimestamp:  time.Now().UTC(),
				Payload:          map[string]any{"selected": "A"},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected sequence gap conflict, got nil")
	}
}

func TestEventIngestion_StaleEpoch_Rejection(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	ctx := context.Background()
	fix, svc := setupTestFixture(t, ctx, pool)

	// Advance attempt session epoch to 2
	repos := repopg.New(pool).Repositories()
	fix.Attempt.SessionEpoch = 2
	if err := repos.Attempts.Update(ctx, &fix.Attempt); err != nil {
		t.Fatalf("failed to update attempt epoch: %v", err)
	}

	// Send event with stale epoch 1
	_, err := svc.SyncEvents(ctx, service.SyncEventsInput{
		AttemptID: fix.Attempt.ID,
		Claims:    fix.Claims,
		Events: []service.SyncEventItemInput{
			{
				EventID:          domain.NewID(),
				SessionID:        &fix.Session.ID,
				SessionEpoch:     1, // Stale epoch (< 2)
				SequenceNumber:   1,
				EventType:        "candidate.option_selected",
				SchemaVersion:    "1.0",
				ClientInstanceID: fix.Session.ClientInstanceID,
				ClientTimestamp:  time.Now().UTC(),
			},
		},
	})
	if !errors.Is(err, domain.ErrStaleEpoch) {
		t.Fatalf("expected ErrStaleEpoch, got: %v", err)
	}
}

func TestEventIngestion_AttemptOwnership_Rejection(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	ctx := context.Background()
	fix, svc := setupTestFixture(t, ctx, pool)

	// Another candidate's claims
	otherSub := domain.NewID()
	otherUser := models.User{
		ID:          domain.NewID(),
		KeycloakSub: otherSub,
		Email:       fmt.Sprintf("other-candidate-%d@example.com", time.Now().UnixNano()),
		Roles:       []string{"candidate"},
	}
	repos := repopg.New(pool).Repositories()
	if err := repos.Users.Upsert(ctx, &otherUser); err != nil {
		t.Fatalf("failed to upsert otherUser: %v", err)
	}
	if err := repos.Candidates.Create(ctx, &models.Candidate{
		ID:       domain.NewID(),
		UserID:   otherUser.ID,
		Status:   domain.CandidateStatusActive,
		FullName: "Other",
		Metadata: map[string]any{},
	}); err != nil {
		t.Fatalf("failed to create otherCandidate: %v", err)
	}

	otherClaims := service.AuthClaims{
		Sub:         otherSub.String(),
		KeycloakSub: otherSub,
		Email:       otherUser.Email,
		Roles:       []string{"candidate"},
	}

	_, err := svc.SyncEvents(ctx, service.SyncEventsInput{
		AttemptID: fix.Attempt.ID,
		Claims:    otherClaims,
		Events: []service.SyncEventItemInput{
			{
				EventID:          domain.NewID(),
				SessionEpoch:     1,
				SequenceNumber:   1,
				EventType:        "candidate.option_selected",
				ClientInstanceID: "other-tab",
				ClientTimestamp:  time.Now().UTC(),
			},
		},
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for non-owned attempt, got: %v", err)
	}
}

func TestEventIngestion_ExpiredAttempt_Rejection(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	ctx := context.Background()
	fix, svc := setupTestFixture(t, ctx, pool)

	// Set expires_at in the past
	past := time.Now().UTC().Add(-10 * time.Minute)
	fix.Attempt.ExpiresAt = &past
	repos := repopg.New(pool).Repositories()
	if err := repos.Attempts.Update(ctx, &fix.Attempt); err != nil {
		t.Fatalf("failed to update attempt: %v", err)
	}

	_, err := svc.SyncEvents(ctx, service.SyncEventsInput{
		AttemptID: fix.Attempt.ID,
		Claims:    fix.Claims,
		Events: []service.SyncEventItemInput{
			{
				EventID:          domain.NewID(),
				SessionEpoch:     1,
				SequenceNumber:   1,
				EventType:        "candidate.option_selected",
				ClientInstanceID: fix.Session.ClientInstanceID,
				ClientTimestamp:  time.Now().UTC(),
			},
		},
	})
	if !errors.Is(err, domain.ErrAttemptExpired) {
		t.Fatalf("expected ErrAttemptExpired, got: %v", err)
	}
}

func TestEventIngestion_FinalizedAttempt_Rejection(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	ctx := context.Background()
	fix, svc := setupTestFixture(t, ctx, pool)

	// Set attempt status to FINALIZED
	fix.Attempt.Status = domain.AttemptStatusFinalized
	repos := repopg.New(pool).Repositories()
	if err := repos.Attempts.Update(ctx, &fix.Attempt); err != nil {
		t.Fatalf("failed to update attempt: %v", err)
	}

	_, err := svc.SyncEvents(ctx, service.SyncEventsInput{
		AttemptID: fix.Attempt.ID,
		Claims:    fix.Claims,
		Events: []service.SyncEventItemInput{
			{
				EventID:          domain.NewID(),
				SessionEpoch:     1,
				SequenceNumber:   1,
				EventType:        "candidate.option_selected",
				ClientInstanceID: fix.Session.ClientInstanceID,
				ClientTimestamp:  time.Now().UTC(),
			},
		},
	})
	if !errors.Is(err, domain.ErrAttemptNotActive) {
		t.Fatalf("expected ErrAttemptNotActive, got: %v", err)
	}
}

func TestEventIngestion_ConcurrentSubmissions_MutualExclusion(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	ctx := context.Background()
	fix, svc := setupTestFixture(t, ctx, pool)

	// Launch 5 concurrent goroutines competing to insert sequence 1
	var wg sync.WaitGroup
	successCount := 0
	conflictCount := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			eventID := domain.NewID()
			_, err := svc.SyncEvents(ctx, service.SyncEventsInput{
				AttemptID: fix.Attempt.ID,
				Claims:    fix.Claims,
				Events: []service.SyncEventItemInput{
					{
						EventID:          eventID,
						SessionID:        &fix.Session.ID,
						SessionEpoch:     1,
						SequenceNumber:   1, // All competing for sequence 1
						EventType:        "candidate.option_selected",
						SchemaVersion:    "1.0",
						ClientInstanceID: fmt.Sprintf("tab-concurrent-%d", idx),
						ClientTimestamp:  time.Now().UTC(),
						Payload:          map[string]any{"idx": idx},
					},
				},
			})
			mu.Lock()
			if err == nil {
				successCount++
			} else {
				conflictCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 concurrent goroutine to succeed, got %d (conflicts: %d)", successCount, conflictCount)
	}
	if conflictCount != 4 {
		t.Fatalf("expected 4 conflicts, got %d", conflictCount)
	}
}
