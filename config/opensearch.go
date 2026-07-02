package config

import (
	"net/http"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/rs/zerolog/log"
)

func (cfg Config) NewOpenSearch() (*opensearch.Client, error) {
	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{
			cfg.OpenSearch.Host,
		},
		Username: cfg.OpenSearch.Username,
		Password: cfg.OpenSearch.Password,
	})

	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.infrastructure.opensearch.NewOpenSearch").
			Msg("failed to initialize opensearch client")

		return nil, err
	}

	res, err := client.Info()
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.infrastructure.opensearch.NewOpenSearch").
			Msg("failed connect to opensearch")

		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Error().
			Int("status_code", res.StatusCode).
			Str("source", "internal.infrastructure.opensearch.NewOpenSearch").
			Msg("opensearch returned non-200 status")

		return nil, err
	}

	return client, nil
}
