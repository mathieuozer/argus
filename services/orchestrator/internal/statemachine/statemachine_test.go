package statemachine

import (
	"testing"
)

func TestCreateTask(t *testing.T) {
	t.Run("creates task in pending state", func(t *testing.T) {
		sm := New()
		task := sm.CreateTask("task-1", "tenant-1", "agent-1", "abc123")

		if task.ID != "task-1" {
			t.Errorf("expected task ID %q, got %q", "task-1", task.ID)
		}
		if task.TenantID != "tenant-1" {
			t.Errorf("expected tenant ID %q, got %q", "tenant-1", task.TenantID)
		}
		if task.AgentID != "agent-1" {
			t.Errorf("expected agent ID %q, got %q", "agent-1", task.AgentID)
		}
		if task.Status != StatusPending {
			t.Errorf("expected status %q, got %q", StatusPending, task.Status)
		}
		if task.InputHash != "abc123" {
			t.Errorf("expected input hash %q, got %q", "abc123", task.InputHash)
		}
		if task.StartedAt.IsZero() {
			t.Error("expected StartedAt to be set")
		}
		if task.CompletedAt != nil {
			t.Error("expected CompletedAt to be nil for new task")
		}
	})
}

func TestTransition(t *testing.T) {
	validTransitionTests := []struct {
		name      string
		from      TaskStatus
		to        TaskStatus
		wantErr   bool
		checkDone bool // whether CompletedAt should be set
	}{
		{
			name:    "pending to running",
			from:    StatusPending,
			to:      StatusRunning,
			wantErr: false,
		},
		{
			name:      "pending to failed",
			from:      StatusPending,
			to:        StatusFailed,
			wantErr:   false,
			checkDone: true,
		},
		{
			name:      "running to completed",
			from:      StatusRunning,
			to:        StatusCompleted,
			wantErr:   false,
			checkDone: true,
		},
		{
			name:      "running to failed",
			from:      StatusRunning,
			to:        StatusFailed,
			wantErr:   false,
			checkDone: true,
		},
		{
			name:    "running to awaiting_approval",
			from:    StatusRunning,
			to:      StatusAwaitingApproval,
			wantErr: false,
		},
		{
			name:    "awaiting_approval to running",
			from:    StatusAwaitingApproval,
			to:      StatusRunning,
			wantErr: false,
		},
		{
			name:      "awaiting_approval to failed",
			from:      StatusAwaitingApproval,
			to:        StatusFailed,
			wantErr:   false,
			checkDone: true,
		},
	}

	for _, tc := range validTransitionTests {
		t.Run(tc.name, func(t *testing.T) {
			sm := New()
			sm.CreateTask("task-1", "tenant-1", "agent-1", "hash")

			// Bring task to the desired starting state
			if tc.from != StatusPending {
				switch tc.from {
				case StatusRunning:
					if err := sm.Transition("task-1", StatusRunning); err != nil {
						t.Fatalf("setup: failed to transition to running: %v", err)
					}
				case StatusAwaitingApproval:
					if err := sm.Transition("task-1", StatusRunning); err != nil {
						t.Fatalf("setup: failed to transition to running: %v", err)
					}
					if err := sm.Transition("task-1", StatusAwaitingApproval); err != nil {
						t.Fatalf("setup: failed to transition to awaiting_approval: %v", err)
					}
				}
			}

			err := sm.Transition("task-1", tc.to)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			task, err := sm.Get("task-1")
			if err != nil {
				t.Fatalf("failed to get task: %v", err)
			}
			if task.Status != tc.to {
				t.Errorf("expected status %q, got %q", tc.to, task.Status)
			}

			if tc.checkDone {
				if task.CompletedAt == nil {
					t.Error("expected CompletedAt to be set for terminal state")
				}
			}
		})
	}

	invalidTransitionTests := []struct {
		name string
		from TaskStatus
		to   TaskStatus
	}{
		{
			name: "pending to completed",
			from: StatusPending,
			to:   StatusCompleted,
		},
		{
			name: "pending to awaiting_approval",
			from: StatusPending,
			to:   StatusAwaitingApproval,
		},
		{
			name: "running to pending",
			from: StatusRunning,
			to:   StatusPending,
		},
		{
			name: "completed to running",
			from: StatusCompleted,
			to:   StatusRunning,
		},
		{
			name: "failed to running",
			from: StatusFailed,
			to:   StatusRunning,
		},
	}

	for _, tc := range invalidTransitionTests {
		t.Run("invalid_"+tc.name, func(t *testing.T) {
			sm := New()
			sm.CreateTask("task-1", "tenant-1", "agent-1", "hash")

			// Bring task to the desired starting state
			switch tc.from {
			case StatusRunning:
				if err := sm.Transition("task-1", StatusRunning); err != nil {
					t.Fatalf("setup: %v", err)
				}
			case StatusCompleted:
				if err := sm.Transition("task-1", StatusRunning); err != nil {
					t.Fatalf("setup: %v", err)
				}
				if err := sm.Transition("task-1", StatusCompleted); err != nil {
					t.Fatalf("setup: %v", err)
				}
			case StatusFailed:
				if err := sm.Transition("task-1", StatusFailed); err != nil {
					t.Fatalf("setup: %v", err)
				}
			}

			err := sm.Transition("task-1", tc.to)
			if err == nil {
				t.Errorf("expected error for transition %s -> %s, got nil", tc.from, tc.to)
			}
		})
	}

	t.Run("transition on nonexistent task", func(t *testing.T) {
		sm := New()
		err := sm.Transition("nonexistent", StatusRunning)
		if err == nil {
			t.Fatal("expected error for nonexistent task, got nil")
		}
	})
}

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		taskID  string
		setup   func(*StateMachine)
		wantErr bool
		wantID  string
	}{
		{
			name:   "found",
			taskID: "task-1",
			setup: func(sm *StateMachine) {
				sm.CreateTask("task-1", "tenant-1", "agent-1", "hash")
			},
			wantErr: false,
			wantID:  "task-1",
		},
		{
			name:    "not found",
			taskID:  "nonexistent",
			setup:   func(sm *StateMachine) {},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := New()
			tc.setup(sm)

			task, err := sm.Get(tc.taskID)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if task.ID != tc.wantID {
				t.Errorf("expected task ID %q, got %q", tc.wantID, task.ID)
			}
		})
	}
}
