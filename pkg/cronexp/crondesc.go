package cronexp

import (
	"github.com/gorhill/cronexpr"
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
	expression = handleSpecialSyntax(expression)

	parsed, err := cronexpr.Parse(expression)
	if err != nil {
		return "", err
	}

	nextTime := parsed.Next(time.Now()).String()
	return nextTime, nil
}

func handleSpecialSyntax(expression string) string {
	switch expression {
	case "@yearly", "@annually":
		return "0 0 0 1 1 * *"
	case "@monthly":
		return "0 0 0 1 * * *"
	case "@weekly":
		return "0 0 0 * * 0 *"
	case "@daily", "@midnight":
		return "0 0 0 * * * *"
	case "@hourly":
		return "0 0 * * * * *"
	default:
		return expression
	}
}
