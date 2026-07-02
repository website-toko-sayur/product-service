package messagingconsumer

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"
)

type ConsumerHandler func(message *sarama.ConsumerMessage) error

type ConsumerGroupHandler struct {
	Handler ConsumerHandler
}

func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			err := h.Handler(message)
			if err != nil {
				log.Error().
					Err(err).
					Str("source", "internal.adapter.message.Consumer.ConsumeClaim").
					Msg("Failed to process message")
			} else {
				session.MarkMessage(message, "")
			}

		case <-session.Context().Done():
			return nil
		}
	}
}

func ConsumeTopic(ctx context.Context, consumerGroup sarama.ConsumerGroup, topic string, handler ConsumerHandler) {
	consumerHandler := &ConsumerGroupHandler{
		Handler: handler,
	}

	go func() {
		for {
			if err := consumerGroup.Consume(ctx, []string{topic}, consumerHandler); err != nil {
				log.Error().
					Err(err).
					Str("source", "internal.adapter.message.Consumer.ConsumeTopic").
					Str("topic", topic).
					Msg("Error from consumer")
			}

			if ctx.Err() != nil {
				return
			}
		}
	}()

	go func() {
		for err := range consumerGroup.Errors() {
			log.Error().
				Err(err).
				Str("source", "internal.adapter.message.Consumer.ConsumeTopic").
				Msg("Consumer group error")
		}
	}()

	<-ctx.Done()
	log.Info().
		Str("topic", topic).
		Str("source", "internal.adapter.message.Consumer.ConsumeTopic").
		Msg("Consumer stopped due to context cancellation")
	if err := consumerGroup.Close(); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.message.Consumer.ConsumeTopic").
			Msg("Error closing consumer group")
	}
}
