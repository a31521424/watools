package launch

import "watools/pkg/models"

type AppScanner interface {
	GetApplications() ([]*models.ApplicationCommand, error)
	RunApplication(path string) error
	ParseApplication(appPath string) (*models.ApplicationCommand, error)
	GetDefaultIconPath() string
}
