package health

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ErrNilDependency is returned if a dependency is missing.
var ErrNilDependency = errors.New("a dependency was expected to be defined but is nil. Please open an issue with the stack trace")

// ExpectDependency expects every dependency to be not nil or it fatals.
func ExpectDependency(logger logrus.FieldLogger, dependencies ...interface{}) {
	if logger == nil {
		panic("missing logger for dependency check")
	}
	for _, d := range dependencies {
		if d == nil {
			logger.WithError(errors.WithStack(ErrNilDependency)).Fatalf("A fatal issue occurred.")
		}
	}
}