package cronexp

import (
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"time"
)

type (
	// Service Cron Descriptor service
	Service interface {
		DescribeCronExpression(expression string) (string, error)
	}

	CronExpression struct {
	}
)

func NewCronExpression() Service {
	return &CronExpression{}
}

func (expr *CronExpression) DescribeCronExpression(expression string) (string, error) {
	log.WithField("expression", expression).Debug("describing cron expression")

	c := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.DowOptional | cron.Descriptor)
	s, err := c.Parse(expression)
	if err != nil {
		return "", err
	}

	st := s.Next(time.Now()).Format(time.RFC3339)
	return st, nil
}
