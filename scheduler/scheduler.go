package scheduler

import (
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/reporting"
)

type Schedule interface {
	WithDateMask(DateMask) Schedule
	PlanForDate(time.Time) []time.Time
	String() string
}

type ScheduledJob struct {
	event     *Event[*ScheduledJob]
	scheduled time.Time
	job       *Job
}

type SchedulerService struct {
	config   *Configuration
	ctx      *appcontext.AppContext
	wg       sync.WaitGroup
	sched    *Scheduler[*ScheduledJob]
	reporter *reporting.Reporter
}

func NewSchedulerService(ctx *appcontext.AppContext, config *Configuration) *SchedulerService {
	return &SchedulerService{
		ctx:    ctx,
		config: config,
		wg:     sync.WaitGroup{},
	}
}

func (s *SchedulerService) Run() {
	s.reporter = reporting.NewReporter(s.ctx)

	runq := make(chan *ScheduledJob, 1000)
	s.sched = NewScheduler(runq)
	stopped, err := s.sched.Start()
	if err != nil {
		s.ctx.GetLogger().Error("failed to start scheduler: %v", err)
		return
	}

	scheduleq := make(chan time.Time, 1)
	sched := NewScheduler(scheduleq)
	_, err = sched.Start()
	if err != nil {
		s.ctx.GetLogger().Error("failed to start scheduler: %v", err)
		return
	}

	t0 := time.Now()
	s.ScheduleForDate(t0)
	sched.ScheduleAt(t0, t0)
	go func() {
		for {
			select {
			case t := <-scheduleq:
				t = s.NextDay(t)
				s.ScheduleForDate(t)
				sched.ScheduleAt(t, t)
			case <-s.ctx.Done():
				sched.Stop()
				s.sched.Stop()
			case <-stopped:
				goto out
			case schedJob := <-runq:
				schedJob.job.Execute(s.ctx, s.reporter, schedJob.scheduled)
			}
		}
	out:
		s.reporter.StopAndWait()
	}()
}

func (s *SchedulerService) NextDay(date time.Time) time.Time {
	year, month, day := date.Date()
	r := time.Date(year, month, day, 0, 0, 0, 0, date.Location())
	return r.AddDate(0, 0, 1)
}

func (s *SchedulerService) ScheduleForDate(date time.Time) {
	s.ctx.GetLogger().Debug("scheduling jobs for %v", date)
	for name, job := range s.config.Jobs {
		for _, schedule := range job.Schedules {
			plan := schedule.PlanForDate(date)
			for _, t := range plan {
				if t.Before(date) {
					s.ctx.GetLogger().Debug("job %q: ignore for past time %v", name, t)
					continue
				}
				s.ctx.GetLogger().Debug("job %q: scheduled for %v", name, t)
				sj := &ScheduledJob{
					scheduled: t,
					job:       job,
				}
				sj.event = s.sched.ScheduleAt(sj, t)
			}
		}
	}
}
