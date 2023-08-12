package steps

import (
	"gopkg.in/errgo.v2/fmt/errors"
)

var ErrMissingClientSettings = errors.Newf("missing client settings")

var ErrMissingClientAPIKey = errors.Newf("missing client settings api key")
