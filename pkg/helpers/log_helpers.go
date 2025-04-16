package helpers

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/lithammer/shortuuid/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Most of this code inspired if not copied from https://github.com/ThreeDotsLabs/go-event-driven

type WatermillZerologAdapter struct {
	logger zerolog.Logger
}

func (w *WatermillZerologAdapter) Error(msg string, err error, fields watermill.LogFields) {
	w.logger.Error().Fields(fields).Err(err).Caller(1).Msg(msg)
}

func (w *WatermillZerologAdapter) Info(msg string, fields watermill.LogFields) {
	// map INFO to DEBUG because watermill is chatty
	w.logger.Debug().Fields(fields).Caller(1).Msg(msg)
}

func (w *WatermillZerologAdapter) Debug(msg string, fields watermill.LogFields) {
	e := w.logger.Debug().Fields(fields)
	e = e.Caller(1)
	e.Msg(msg)
}

func (w *WatermillZerologAdapter) Trace(msg string, fields watermill.LogFields) {
	w.logger.Trace().Fields(fields).Caller(1).Msg(msg)
}

func (w *WatermillZerologAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	l := w.logger.With().Fields(fields).Logger()
	return &WatermillZerologAdapter{logger: l}
}

func NewWatermill(logger zerolog.Logger) *WatermillZerologAdapter {
	return &WatermillZerologAdapter{logger: logger}
}

var _ watermill.LoggerAdapter = &WatermillZerologAdapter{}

const correlationIDMessageMetadataKey = "correlation_id"

type CorrelationPublisherDecorator struct {
	message.Publisher
}

type correlationIDKeyType string

const correlationIDKey correlationIDKeyType = "correlation_id"

func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

func CorrelationIDFromContext(ctx context.Context) string {
	v, ok := ctx.Value(correlationIDKey).(string)
	if ok {
		return v
	}

	log.Ctx(ctx).Warn().Msg("correlation ID not found in context")

	// add "gen_" prefix to distinguish generated correlation IDs from correlation IDs passed by the client
	// it's useful to detect if correlation ID was not passed properly
	return "gen_" + shortuuid.New()
}

func (c CorrelationPublisherDecorator) Publish(topic string, messages ...*message.Message) error {
	for i := range messages {
		// if correlation_id is already set, let's not override
		if messages[i].Metadata.Get(correlationIDMessageMetadataKey) != "" {
			continue
		}

		// correlation_id as const
		messages[i].Metadata.Set(correlationIDMessageMetadataKey, CorrelationIDFromContext(messages[i].Context()))
	}

	return c.Publisher.Publish(topic, messages...)
}
