package steps

import (
	"github.com/pkg/errors"
)

var ErrMissingClientSettings = errors.New("missing client settings")

var ErrMissingClientAPIKey = errors.New("missing client settings api key")
