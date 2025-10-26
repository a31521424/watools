package models

import (
	"time"

	"github.com/samber/mo"
)

type OptionTime = mo.Option[time.Time]

func ToOptionTime(t time.Time) OptionTime {
	return mo.Some(t)
}
