package application

import "watools/pkg/models"

type AppScanner interface {
	GetApplications() ([]*models.ApplicationCommand, error)
	ParseApplication(appPath string) (*models.ApplicationCommand, error)
	GetDefaultIconPath() string
}
