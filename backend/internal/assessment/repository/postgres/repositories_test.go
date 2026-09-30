package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentinelexam/backend/internal/assessment/domain"
	"github.com/sentinelexam/backend/internal/assessment/models"
	repopg "github.com/sentinelexam/backend/internal/assessment/repository/postgres"
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
		t.Skipf("skipping postgres integration test: failed to connect to %s: %v", dsn, err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping postgres integration test: ping failed: %v", err)
	}
	return pool
}

// createTestAttempt creates a dummy exam, candidate, assignment and attempt for testing foreign keys.
func createTestAttempt(t *testing.T, ctx context.Context, pool *pgxpool.Pool) domain.ID {
	t.Helper()
	db := repopg.New(pool)
	repos := db.Repositories()

	// 1. User
	user := &models.User{
		ID:          domain.NewID(),
		KeycloakSub: domain.NewID(),
		Email:       fmt.Sprintf("test-resilience-%s@example.com", uuid.New().String()[:8]),
		DisplayName: "Resilience Test User",
		Roles:       []string{"candidate"},
	}
	if err := repos.Users.Upsert(ctx, user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// 2. Candidate
	cand := &models.Candidate{
		ID:       domain.NewID(),
		UserID:   user.ID,
		Status:   domain.CandidateStatusActive,
		FullName: "Resilience Candidate",
		Metadata: map[string]any{},
	}
	if err := repos.Candidates.Create(ctx, cand); err != nil {
		t.Fatalf("failed to create candidate: %v", err)
	}

	// 3. Exam
	exam := &models.Exam{
		ID:              domain.NewID(),
		Title:           "Resilience Test Exam",
		Description:     "Exam for testing Phase 4 resilience event WAL",
		Status:          domain.ExamStatusLive,
		DurationSeconds: 3600,
		MaxAttempts:     1,
		Navigation:      map[string]any{"allow_backtracking": true},
		Scoring:         map[string]any{"mode": "STANDARD"},
		Version:         1,
	}
	if err := repos.Exams.Create(ctx, exam); err != nil {
		t.Fatalf("failed to create exam: %v", err)
	}

	// 4. Assignment
	assign := &models.ExamAssignment{
		ID:          domain.NewID(),
		ExamID:      exam.ID,
		CandidateID: cand.ID,
		Status:      domain.AssignmentStatusAssigned,
		AssignedAt:  time.Now().UTC(),
	}
	if err := repos.Assignments.Create(ctx, assign); err != nil {
		t.Fatalf("failed to create assignment: %v", err)
	}

	// 5. Attempt
	now := time.Now().UTC()
	exp := now.Add(1 * time.Hour)
	attempt := &models.ExamAttempt{
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
	if err := repos.Attempts.Create(ctx, attempt); err != nil {
		t.Fatalf("failed to create attempt: %v", err)
	}

	return attempt.ID
}

func TestCandidateEventRepository_SequentialInsertAndQuery(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()
	attemptID := createTestAttempt(t, ctx, pool)

	db := repopg.New(pool)
	repos := db.Repositories()
	eventRepo := repos.CandidateEvents

	// 1. Initial max sequence should be 0
	maxSeq, err := eventRepo.GetMaxSequence(ctx, attemptID)
	if err != nil {
		t.Fatalf("failed to get max sequence: %v", err)
	}
	if maxSeq != 0 {
		t.Fatalf("expected initial max sequence 0, got %d", maxSeq)
	}

	// 2. Insert 5 sequential candidate events
	now := time.Now().UTC().Truncate(time.Microsecond)
	var createdEvents []models.CandidateEvent
	for seq := int64(1); seq <= 5; seq++ {
		eventID := domain.NewID()
		prevH := fmt.Sprintf("prev_hash_%d", seq-1)
		evtH := fmt.Sprintf("event_hash_%d", seq)
		evt := models.CandidateEvent{
			ID:               domain.NewID(),
			EventID:          eventID,
			AttemptID:        attemptID,
			SessionEpoch:     1,
			SequenceNumber:   seq,
			EventType:        domain.CandidateEventTypeOptionSelected,
			SchemaVersion:    "1.0",
			ClientInstanceID: "tab-inst-resilience-1",
			Payload: map[string]any{
				"question_idx": seq,
				"selected":     "B",
			},
			ClientTimestamp:  now.Add(time.Duration(seq) * time.Second),
			ServerReceivedAt: now.Add(time.Duration(seq)*time.Second + 50*time.Millisecond),
			PrevHash:         &prevH,
			EventHash:        &evtH,
			ProcessingStatus: domain.EventProcessingStatusCommitted,
		}

		if err := eventRepo.Insert(ctx, &evt); err != nil {
			t.Fatalf("failed to insert candidate event %d: %v", seq, err)
		}
		createdEvents = append(createdEvents, evt)
	}

	// 3. Verify max sequence is now 5
	maxSeq, err = eventRepo.GetMaxSequence(ctx, attemptID)
	if err != nil {
		t.Fatalf("failed to get max sequence: %v", err)
	}
	if maxSeq != 5 {
		t.Fatalf("expected max sequence 5, got %d", maxSeq)
	}

	// 4. FindByEventID
	for _, expected := range createdEvents {
		fetched, err := eventRepo.FindByEventID(ctx, expected.EventID)
		if err != nil {
			t.Fatalf("failed to find by event_id %s: %v", expected.EventID, err)
		}
		if fetched.EventID != expected.EventID {
			t.Fatalf("event_id mismatch: got %s, want %s", fetched.EventID, expected.EventID)
		}
		if fetched.SequenceNumber != expected.SequenceNumber {
			t.Fatalf("sequence_number mismatch: got %d, want %d", fetched.SequenceNumber, expected.SequenceNumber)
		}
		if fetched.ClientInstanceID != expected.ClientInstanceID {
			t.Fatalf("client_instance_id mismatch: got %s, want %s", fetched.ClientInstanceID, expected.ClientInstanceID)
		}
		if fetched.EventType != expected.EventType {
			t.Fatalf("event_type mismatch: got %s, want %s", fetched.EventType, expected.EventType)
		}
	}

	// 5. FindByAttemptAndSequence
	for _, expected := range createdEvents {
		fetched, err := eventRepo.FindByAttemptAndSequence(ctx, attemptID, expected.SequenceNumber)
		if err != nil {
			t.Fatalf("failed to find by attempt %s and seq %d: %v", attemptID, expected.SequenceNumber, err)
		}
		if fetched.EventID != expected.EventID {
			t.Fatalf("event_id mismatch: got %s, want %s", fetched.EventID, expected.EventID)
		}
	}

	// 6. ListByAttempt - verify strictly ascending sequence order
	allEvents, err := eventRepo.ListByAttempt(ctx, attemptID)
	if err != nil {
		t.Fatalf("failed to list events by attempt: %v", err)
	}
	if len(allEvents) != 5 {
		t.Fatalf("expected 5 events, got %d", len(allEvents))
	}
	for i, e := range allEvents {
		expectedSeq := int64(i + 1)
		if e.SequenceNumber != expectedSeq {
			t.Fatalf("event at index %d has seq %d, want %d", i, e.SequenceNumber, expectedSeq)
		}
	}

	// 7. ListByAttemptRange (e.g. seq 2 to 4)
	rangeEvents, err := eventRepo.ListByAttemptRange(ctx, attemptID, 2, 4)
	if err != nil {
		t.Fatalf("failed to list events by range: %v", err)
	}
	if len(rangeEvents) != 3 {
		t.Fatalf("expected 3 events in range [2,4], got %d", len(rangeEvents))
	}
	if rangeEvents[0].SequenceNumber != 2 || rangeEvents[1].SequenceNumber != 3 || rangeEvents[2].SequenceNumber != 4 {
		t.Fatalf("unexpected range sequence numbers: %v", rangeEvents)
	}
}

func TestCandidateEventRepository_DuplicateEventIDRejection(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()
	attemptID := createTestAttempt(t, ctx, pool)

	db := repopg.New(pool)
	eventRepo := db.Repositories().CandidateEvents

	sharedEventID := domain.NewID()
	evt1 := models.CandidateEvent{
		ID:               domain.NewID(),
		EventID:          sharedEventID,
		AttemptID:        attemptID,
		SessionEpoch:     1,
		SequenceNumber:   1,
		EventType:        domain.CandidateEventTypeOptionSelected,
		ClientInstanceID: "tab-1",
		ClientTimestamp:  time.Now().UTC(),
	}

	if err := eventRepo.Insert(ctx, &evt1); err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	// Attempt inserting another event with the same event_id but different sequence
	evt2 := models.CandidateEvent{
		ID:               domain.NewID(),
		EventID:          sharedEventID,
		AttemptID:        attemptID,
		SessionEpoch:     1,
		SequenceNumber:   2,
		EventType:        domain.CandidateEventTypeTextAnswered,
		ClientInstanceID: "tab-1",
		ClientTimestamp:  time.Now().UTC(),
	}

	err := eventRepo.Insert(ctx, &evt2)
	if err == nil {
		t.Fatalf("expected duplicate event_id insert to fail with unique constraint violation, got nil")
	}
}

func TestCandidateEventRepository_DuplicateAttemptSequenceRejection(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()
	attemptID := createTestAttempt(t, ctx, pool)

	db := repopg.New(pool)
	eventRepo := db.Repositories().CandidateEvents

	evt1 := models.CandidateEvent{
		ID:               domain.NewID(),
		EventID:          domain.NewID(),
		AttemptID:        attemptID,
		SessionEpoch:     1,
		SequenceNumber:   1,
		EventType:        domain.CandidateEventTypeOptionSelected,
		ClientInstanceID: "tab-1",
		ClientTimestamp:  time.Now().UTC(),
	}

	if err := eventRepo.Insert(ctx, &evt1); err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	// Attempt inserting another event with DIFFERENT event_id but SAME sequence_number
	evt2 := models.CandidateEvent{
		ID:               domain.NewID(),
		EventID:          domain.NewID(),
		AttemptID:        attemptID,
		SessionEpoch:     1,
		SequenceNumber:   1, // Duplicate attempt_id + sequence_number
		EventType:        domain.CandidateEventTypeOptionSelected,
		ClientInstanceID: "tab-2",
		ClientTimestamp:  time.Now().UTC(),
	}

	err := eventRepo.Insert(ctx, &evt2)
	if err == nil {
		t.Fatalf("expected duplicate (attempt_id, sequence_number) insert to fail, got nil")
	}
}

func TestCandidateEventAndOutbox_TransactionalAtomicity(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()
	attemptID := createTestAttempt(t, ctx, pool)

	// 1. Test successful atomic commit of CandidateEvent + OutboxEvent in single pgx.Tx
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	eventRepoTx := repopg.NewCandidateEventRepoTx(tx)
	outboxRepoTx := repopg.NewOutboxRepoTx(tx)

	eventID := domain.NewID()
	outboxID := domain.NewID()

	candEvent := &models.CandidateEvent{
		ID:               domain.NewID(),
		EventID:          eventID,
		AttemptID:        attemptID,
		SessionEpoch:     1,
		SequenceNumber:   1,
		EventType:        domain.CandidateEventTypeOptionSelected,
		ClientInstanceID: "tab-tx-1",
		Payload:          map[string]any{"selected": "A"},
		ClientTimestamp:  time.Now().UTC(),
	}
	if err := eventRepoTx.Insert(ctx, candEvent); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("failed to insert candidate event in tx: %v", err)
	}

	outboxEvent := &models.OutboxEvent{
		ID:         outboxID,
		EventType:  string(domain.CandidateEventTypeOptionSelected),
		StreamName: "SENTINEL_CANDIDATE_EVENTS",
		Subject:    "sentinel.candidate.option_selected",
		Payload: map[string]any{
			"event_id":   eventID.String(),
			"attempt_id": attemptID.String(),
			"sequence":   1,
		},
		Status: domain.OutboxStatusPending,
	}
	if err := outboxRepoTx.Create(ctx, outboxEvent); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("failed to insert outbox event in tx: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("failed to commit tx: %v", err)
	}

	// 2. Verify committed data is readable via pool repos
	db := repopg.New(pool)
	repos := db.Repositories()

	savedEvt, err := repos.CandidateEvents.FindByEventID(ctx, eventID)
	if err != nil {
		t.Fatalf("committed candidate event not found: %v", err)
	}
	if savedEvt.SequenceNumber != 1 {
		t.Fatalf("expected sequence 1, got %d", savedEvt.SequenceNumber)
	}

	pendingOutbox, err := repos.OutboxEvents.ListPending(ctx, 1000)
	if err != nil {
		t.Fatalf("failed to list pending outbox: %v", err)
	}
	var foundOutbox bool
	for _, o := range pendingOutbox {
		if o.ID == outboxID {
			foundOutbox = true
			if o.Status != domain.OutboxStatusPending {
				t.Fatalf("expected status PENDING, got %s", o.Status)
			}
			break
		}
	}
	if !foundOutbox {
		t.Fatalf("committed outbox event %s not found in pending list", outboxID)
	}

	// 3. Mark published test
	if err := repos.OutboxEvents.MarkPublished(ctx, outboxID); err != nil {
		t.Fatalf("failed to mark outbox published: %v", err)
	}

	// 4. Test Transaction Rollback (Neither event should persist)
	txRollback, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		t.Fatalf("failed to begin rollback tx: %v", err)
	}

	rbEventRepo := repopg.NewCandidateEventRepoTx(txRollback)
	rbOutboxRepo := repopg.NewOutboxRepoTx(txRollback)

	rbEventID := domain.NewID()
	rbOutboxID := domain.NewID()

	_ = rbEventRepo.Insert(ctx, &models.CandidateEvent{
		ID:               domain.NewID(),
		EventID:          rbEventID,
		AttemptID:        attemptID,
		SessionEpoch:     1,
		SequenceNumber:   2,
		EventType:        domain.CandidateEventTypeHeartbeat,
		ClientInstanceID: "tab-tx-rollback",
		ClientTimestamp:  time.Now().UTC(),
	})
	_ = rbOutboxRepo.Create(ctx, &models.OutboxEvent{
		ID:         rbOutboxID,
		EventType:  string(domain.CandidateEventTypeHeartbeat),
		StreamName: "SENTINEL_CANDIDATE_EVENTS",
		Subject:    "sentinel.candidate.heartbeat",
		Payload:    map[string]any{"event_id": rbEventID.String()},
	})

	// Rollback transaction
	if err := txRollback.Rollback(ctx); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	// Verify neither was persisted
	_, err = repos.CandidateEvents.FindByEventID(ctx, rbEventID)
	if err != domain.ErrNotFound {
		t.Fatalf("expected domain.ErrNotFound after rollback, got %v", err)
	}
}
