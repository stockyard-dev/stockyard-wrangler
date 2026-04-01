package server

import "github.com/stockyard-dev/stockyard-wrangler/internal/license"

type Limits struct {
	MaxQueues      int  // 0 = unlimited
	MaxJobsPerMonth int // 0 = unlimited
	MaxAttempts    int  // free: 3, pro: unlimited
	Scheduling     bool // run_at support (Pro)
	PriorityJobs   bool // Pro
	WebhookOnFail  bool // Pro
	RetentionDays  int
}

var freeLimits = Limits{
	MaxQueues:       1,
	MaxJobsPerMonth: 1000,
	MaxAttempts:     3,
	Scheduling:      false,
	PriorityJobs:    false,
	WebhookOnFail:   false,
	RetentionDays:   7,
}

var proLimits = Limits{
	MaxQueues:       0,
	MaxJobsPerMonth: 0,
	MaxAttempts:     0,
	Scheduling:      true,
	PriorityJobs:    true,
	WebhookOnFail:   true,
	RetentionDays:   90,
}

func LimitsFor(info *license.Info) Limits {
	if info != nil && info.IsPro() {
		return proLimits
	}
	return freeLimits
}

func LimitReached(limit, current int) bool {
	if limit == 0 {
		return false
	}
	return current >= limit
}
