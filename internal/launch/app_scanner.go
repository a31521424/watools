package launch

import "watools/pkg/models"

type AppScanner interface {
	GetApplications() ([]*models.Command, error)
	RunApplication(path string) error
	ParseApplication(appPath string) (*models.Command, error)
}
