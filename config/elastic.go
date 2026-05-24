package config

import (
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/rs/zerolog/log"
)

func (cfg Config) NewElastic() (*elasticsearch.Client, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			cfg.ElasticSearch.Host,
		},
		// Username: cfg.ElasticSearch.Username,
		// Password: cfg.ElasticSearch.Password,
	})

	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.infrastructure.elasticsearch.NewElastic").
			Msg("failed to initialize elasticsearch client")

		return nil, err
	}

	info, err := es.Info()
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.infrastructure.elasticsearch.NewElastic").
			Msg("failed to connect to elasticsearch")

		return nil, err
	}

	defer info.Body.Close()

	log.Info().
		Str("source", "internal.infrastructure.elasticsearch.NewElastic").
		Msg("success connect to elasticsearch")

	return es, nil
}
