package reporting

import "context"

type NullEmitter struct {
}

func (emitter *NullEmitter) Emit(ctx context.Context, report *Report) error {
	return nil
}
