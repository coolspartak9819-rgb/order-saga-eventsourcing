package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/coolspartak9819-rgb/queuelens/internal/domain"
	"github.com/coolspartak9819-rgb/queuelens/internal/events"
	"github.com/coolspartak9819-rgb/queuelens/internal/queue"
	"github.com/coolspartak9819-rgb/queuelens/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	jobStore, err := store.New(ctx, env("DATABASE_URL", "postgres://queuelens:queuelens@localhost:5432/queuelens"))
	if err != nil {
		log.Fatal(err)
	}
	defer jobStore.Close()
	jobQueue := queue.New(env("REDIS_ADDR", "localhost:6379"), env("QUEUE_STREAM", "jobs"), env("WORKER_GROUP", "queuelens-workers"), env("WORKER_NAME", "worker-1"))
	defer jobQueue.Close()
	publisher, err := events.NewPublisher(env("KAFKA_BROKERS", "localhost:9092"), env("KAFKA_TOPIC", "job-events"))
	if err != nil {
		log.Fatal(err)
	}
	defer publisher.Close()
	if err := jobQueue.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	if err := jobQueue.EnsureGroup(ctx); err != nil {
		log.Fatal(err)
	}
	maxAttempts, _ := strconv.Atoi(env("MAX_ATTEMPTS", "3"))
	if maxAttempts < 1 {
		maxAttempts = 3
	}
	log.Println("QueueLens worker started")
	for {
		if err := process(ctx, jobStore, jobQueue, publisher, maxAttempts); errors.Is(err, context.Canceled) {
			return
		} else if err != nil {
			log.Printf("worker error: %v", err)
			time.Sleep(time.Second)
		}
	}
}

func process(ctx context.Context, jobStore *store.Postgres, jobQueue *queue.Redis, publisher *events.Publisher, maxAttempts int) error {
	message, err := jobQueue.Next(ctx)
	if errors.Is(err, queue.ErrNoMessage) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := jobStore.SetRunning(ctx, message.JobID); err != nil {
		return err
	}
	if string(message.Payload) == "{\"fail\":true}" {
		return failJob(ctx, jobStore, jobQueue, publisher, message, maxAttempts, errors.New("simulated job failure"))
	}
	time.Sleep(300 * time.Millisecond)
	if err := jobStore.Complete(ctx, message.JobID); err != nil {
		return err
	}
	if err := publisher.Publish(ctx, events.JobEvent{JobID: message.JobID, Type: message.Type, Status: domain.StatusCompleted, OccurredAt: time.Now().UTC()}); err != nil {
		log.Printf("publish completed event: %v", err)
	}
	return jobQueue.Ack(ctx, message.ID)
}

func failJob(ctx context.Context, jobStore *store.Postgres, jobQueue *queue.Redis, publisher *events.Publisher, message queue.Message, maxAttempts int, cause error) error {
	job, err := jobStore.Get(ctx, message.JobID)
	if err != nil {
		return err
	}
	if job.Attempts >= maxAttempts {
		if err := jobStore.Fail(ctx, message.JobID, cause.Error()); err != nil {
			return err
		}
		if err := jobQueue.DeadLetter(ctx, message, cause.Error()); err != nil {
			return err
		}
		if err := publisher.Publish(ctx, events.JobEvent{JobID: message.JobID, Type: message.Type, Status: domain.StatusFailed, Attempts: job.Attempts, Error: cause.Error(), OccurredAt: time.Now().UTC()}); err != nil {
			log.Printf("publish failed event: %v", err)
		}
		return jobQueue.Ack(ctx, message.ID)
	}
	if err := jobStore.Retry(ctx, message.JobID, cause.Error()); err != nil {
		return err
	}
	if err := publisher.Publish(ctx, events.JobEvent{JobID: message.JobID, Type: message.Type, Status: domain.StatusRetrying, Attempts: job.Attempts, Error: cause.Error(), OccurredAt: time.Now().UTC()}); err != nil {
		log.Printf("publish retry event: %v", err)
	}
	if err := jobQueue.Ack(ctx, message.ID); err != nil {
		return err
	}
	time.Sleep(time.Duration(job.Attempts) * time.Second)
	return jobQueue.Enqueue(ctx, messageToJob(message))
}

func messageToJob(message queue.Message) domain.Job {
	return domain.Job{ID: message.JobID, Type: message.Type, Payload: message.Payload, Status: domain.StatusRetrying}
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
