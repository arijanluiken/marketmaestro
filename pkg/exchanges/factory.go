package exchanges

import (
	"fmt"

	"github.com/rs/zerolog"
)

// Factory implements the exchanges.Factory interface
type Factory struct {
	logger zerolog.Logger
}

// NewFactory creates a new exchange factory
func NewFactory(logger zerolog.Logger) *Factory {
	return &Factory{
		logger: logger.With().Str("component", "exchange_factory").Logger(),
	}
}

// CreateExchange creates an exchange instance based on the exchange name
func (f *Factory) CreateExchange(exchangeName string, config map[string]interface{}) (Exchange, error) {
	f.logger.Info().
		Str("exchange", exchangeName).
		Msg("Creating exchange instance")

	switch exchangeName {
	case "bybit":
		return f.createBybitExchange(config)
	case "bitvavo":
		return f.createBitvavoExchange(config)
	default:
		return nil, fmt.Errorf("unsupported exchange: %s", exchangeName)
	}
}

// GetSupportedExchanges returns a list of supported exchange names
func (f *Factory) GetSupportedExchanges() []string {
	return []string{"bybit", "bitvavo"}
}

func (f *Factory) createBybitExchange(config map[string]interface{}) (Exchange, error) {
	apiKey, ok := config["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("bybit api_key is required")
	}

	secret, ok := config["secret"].(string)
	if !ok || secret == "" {
		return nil, fmt.Errorf("bybit secret is required")
	}

	testnet, _ := config["testnet"].(bool)

	return NewBybit(apiKey, secret, testnet, f.logger), nil
}

func (f *Factory) createBitvavoExchange(config map[string]interface{}) (Exchange, error) {
	apiKey, ok := config["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("bitvavo api_key is required")
	}

	secret, ok := config["secret"].(string)
	if !ok || secret == "" {
		return nil, fmt.Errorf("bitvavo secret is required")
	}

	testnet, _ := config["testnet"].(bool)

	return NewBitvavo(apiKey, secret, testnet, f.logger), nil
}