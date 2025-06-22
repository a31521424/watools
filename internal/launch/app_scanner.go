package launch

import "watools/pkg/models"

type AppScanner interface {
	GetApplication() ([]models.Command, error)
	RunApplication(path string) error
}
