package validator

import (
	"errors"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"

	ut "github.com/go-playground/universal-translator"
)

type Validator struct {
	Validator  *validator.Validate
	Translator ut.Translator
}

func NewValidator() *Validator {
	en := en.New()
	uni := ut.New(en, en)
	trans, found := uni.GetTranslator("en")
	if !found {
		log.Error().
			Str("source", "utils.validator.NewValidator").
			Msg("Translator not found")
	}

	validate := validator.New()

	return &Validator{
		Validator:  validate,
		Translator: trans,
	}
}

func (v *Validator) Validate(i any) error {
	err := v.Validator.Struct(i)

	if err != nil {
		validationErrors, ok := err.(validator.ValidationErrors)

		if ok {
			for _, e := range validationErrors {
				log.Warn().
					Str("source", "utils.validator.Validate").
					Str("field", e.Field()).
					Str("tag", e.Tag()).
					Msg("Validation failed")

				return errors.New(e.Translate(v.Translator))
			}
		}

		log.Error().
			Err(err).
			Str("source", "utils.validator.Validate").
			Msg("Failed to validate request")

		return err
	}

	return nil
}
