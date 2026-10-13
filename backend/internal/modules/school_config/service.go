package school_config

import "elimu-core/backend/internal/domain"

type Service = domain.Service

func NewService(settings SchoolSettings, notifier domain.Notifier) (*Service, error) {
	return domain.NewService(settings, notifier)
}
