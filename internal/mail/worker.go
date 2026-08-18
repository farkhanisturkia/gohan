package mail

import (
	"fmt"
	"log"
)

type EmailJob struct {
	To      string
	Subject string
	Body    string
}

var emailQueue = make(chan EmailJob, 1000)

func InitMailWorker(workerCount int) {
	for i := 1; i <= workerCount; i++ {
		go func(workerID int) {
			for job := range emailQueue {
				if err := SendEmail(job.To, job.Subject, job.Body); err != nil {
					log.Printf("[MAIL WORKER %d ERROR] Failed to send email to %s: %v\n", workerID, job.To, err)
				} else {
					log.Printf("[MAIL WORKER %d SUCCESS] Email sent to %s\n", workerID, job.To)
				}
			}
		}(i)
	}
	log.Printf("[info] Mail worker pool started with %d workers\n", workerCount)
}

func QueueEmail(to, subject, body string) error {
	select {
	case emailQueue <- EmailJob{To: to, Subject: subject, Body: body}:
		return nil
	default:
		return fmt.Errorf("email queue is full")
	}
}