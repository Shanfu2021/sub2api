package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// Task type constants
const (
	TaskTypeVerifyCode    = "verify_code"
	TaskTypePasswordReset = "password_reset"
)

const (
	emailQueueMaxAttempts     = 3
	emailQueueAttemptTimeout   = 20 * time.Second
	emailQueueRetryBackoffBase = 2 * time.Second
)

// EmailTask 邮件发送任务
type EmailTask struct {
	Email    string
	SiteName string
	TaskType string // "verify_code" or "password_reset"
	ResetURL string // Only used for password_reset task type
	Locale   string // Optional Accept-Language locale hint
}

// EmailQueueService 异步邮件队列服务
type EmailQueueService struct {
	emailService *EmailService
	taskChan     chan EmailTask
	wg           sync.WaitGroup
	stopChan     chan struct{}
	workers      int
}

// NewEmailQueueService 创建邮件队列服务
func NewEmailQueueService(emailService *EmailService, workers int) *EmailQueueService {
	if workers <= 0 {
		workers = 3 // 默认3个工作协程
	}

	service := &EmailQueueService{
		emailService: emailService,
		taskChan:     make(chan EmailTask, 100), // 缓冲100个任务
		stopChan:     make(chan struct{}),
		workers:      workers,
	}

	// 启动工作协程
	service.start()

	return service
}

// start 启动工作协程
func (s *EmailQueueService) start() {
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
	logger.LegacyPrintf("service.email_queue", "[EmailQueue] Started %d workers", s.workers)
}

// worker 工作协程
func (s *EmailQueueService) worker(id int) {
	defer s.wg.Done()

	for {
		select {
		case task := <-s.taskChan:
			s.processTask(id, task)
		case <-s.stopChan:
			logger.LegacyPrintf("service.email_queue", "[EmailQueue] Worker %d stopping", id)
			return
		}
	}
}

// processTask 处理任务
func (s *EmailQueueService) processTask(workerID int, task EmailTask) {
	for attempt := 1; attempt <= emailQueueMaxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), emailQueueAttemptTimeout)
		err := s.processTaskAttempt(ctx, task, attempt)
		cancel()

		if err == nil {
			if attempt == 1 {
				logger.LegacyPrintf("service.email_queue", "[EmailQueue] Worker %d sent %s to %s", workerID, task.TaskType, task.Email)
			} else {
				logger.LegacyPrintf("service.email_queue", "[EmailQueue] Worker %d sent %s to %s after %d attempts", workerID, task.TaskType, task.Email, attempt)
			}
			return
		}

		if attempt >= emailQueueMaxAttempts || !isRetryableEmailError(err) {
			logger.LegacyPrintf("service.email_queue", "[EmailQueue] Worker %d failed to send %s to %s: %v", workerID, task.TaskType, task.Email, err)
			return
		}

		backoff := time.Duration(attempt) * emailQueueRetryBackoffBase
		logger.LegacyPrintf("service.email_queue", "[EmailQueue] Worker %d retrying %s to %s in %s after error: %v", workerID, task.TaskType, task.Email, backoff, err)

		timer := time.NewTimer(backoff)
		select {
		case <-timer.C:
		case <-s.stopChan:
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

func (s *EmailQueueService) processTaskAttempt(ctx context.Context, task EmailTask, attempt int) error {
	switch task.TaskType {
	case TaskTypeVerifyCode:
		if attempt == 1 {
			return s.emailService.SendVerifyCode(ctx, task.Email, task.SiteName, task.Locale)
		}
		return s.emailService.ResendLatestVerifyCode(ctx, task.Email, task.SiteName)
	case TaskTypePasswordReset:
		if attempt == 1 {
			return s.emailService.SendPasswordResetEmailWithCooldown(ctx, task.Email, task.SiteName, task.ResetURL, task.Locale)
		}
		return s.emailService.SendPasswordResetEmail(ctx, task.Email, task.SiteName, task.ResetURL, task.Locale)
	default:
		return fmt.Errorf("unknown task type: %s", task.TaskType)
	}
}

func isRetryableEmailError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	msg := strings.ToLower(err.Error())
	for _, needle := range []string{
		"timeout",
		"temporary",
		"connection reset",
		"broken pipe",
		"unexpected eof",
		"tls dial",
		"i/o timeout",
		"421 ",
		"450 ",
		"451 ",
		"452 ",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// EnqueueVerifyCode 将验证码发送任务加入队列
func (s *EmailQueueService) EnqueueVerifyCode(email, siteName string, locale ...string) error {
	task := EmailTask{
		Email:    email,
		SiteName: siteName,
		TaskType: TaskTypeVerifyCode,
		Locale:   firstEmailLocale(locale),
	}

	select {
	case s.taskChan <- task:
		logger.LegacyPrintf("service.email_queue", "[EmailQueue] Enqueued verify code task for %s", email)
		return nil
	default:
		return fmt.Errorf("email queue is full")
	}
}

// EnqueuePasswordReset 将密码重置邮件任务加入队列
func (s *EmailQueueService) EnqueuePasswordReset(email, siteName, resetURL string, locale ...string) error {
	task := EmailTask{
		Email:    email,
		SiteName: siteName,
		TaskType: TaskTypePasswordReset,
		ResetURL: resetURL,
		Locale:   firstEmailLocale(locale),
	}

	select {
	case s.taskChan <- task:
		logger.LegacyPrintf("service.email_queue", "[EmailQueue] Enqueued password reset task for %s", email)
		return nil
	default:
		return fmt.Errorf("email queue is full")
	}
}

// Stop 停止队列服务
func (s *EmailQueueService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
	logger.LegacyPrintf("service.email_queue", "%s", "[EmailQueue] All workers stopped")
}
