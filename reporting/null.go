package reporting

import (
	"github.com/PlakarKorp/plakar/logging"
)

type NullEmitter struct {
}

func (emitter *NullEmitter) Emit(report Report, logger *logging.Logger) {
}
