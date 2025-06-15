package apps

import (
	"context"
	"errors"
	"runtime"
	"watools/schemas"
)

// TODO: optimize

type Platform interface {
	GetApplication() (schemas.CommandGroup, error)
}

type BasePlatform struct {
	Platform
	ctx context.Context
}

func NewBasePlatform(ctx context.Context) *BasePlatform {
	return &BasePlatform{
		ctx: ctx,
	}
}

type Mac struct {
	BasePlatform
}

func (m *Mac) GetApplication() (schemas.CommandGroup, error) {
	commands := GetMacApplication()
	return schemas.CommandGroup{
		Category: schemas.CategoryApplication,
		Commands: commands,
	}, nil
}

type Windows struct {
	BasePlatform
}

func (w *Windows) GetApplication() (schemas.CommandGroup, error) {
	return schemas.CommandGroup{}, errors.New("windows not implemented")
}

type Linux struct {
	BasePlatform
}

func (l *Linux) GetApplication() (schemas.CommandGroup, error) {
	return schemas.CommandGroup{}, errors.New("linux not implemented")
}

func NewPlatform(ctx context.Context) (Platform, error) {
	base := NewBasePlatform(ctx)
	switch runtime.GOOS {
	case "darwin":
		return &Mac{
			BasePlatform: *base,
		}, nil

	case "windows":
		return &Windows{
			BasePlatform: *base,
		}, nil

	case "linux":
		return &Linux{
			BasePlatform: *base,
		}, nil
	}
	return nil, errors.New("unknown platform")
}
