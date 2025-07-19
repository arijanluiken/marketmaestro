package order

import (
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
)

// Messages for order manager actor communication
type (
	PlaceOrderMsg   struct{}
	CancelOrderMsg  struct{}
	GetOrdersMsg    struct{}
	StatusMsg       struct{}
)

// OrderManagerActor manages order placement and advanced order types
type OrderManagerActor struct {
	exchangeName string
	config       *config.Config
	db           *database.DB
	logger       zerolog.Logger
	orders       map[string]interface{} // Active orders
}

// New creates a new order manager actor
func New(exchangeName string, cfg *config.Config, db *database.DB, logger zerolog.Logger) *OrderManagerActor {
	return &OrderManagerActor{
		exchangeName: exchangeName,
		config:       cfg,
		db:           db,
		logger:       logger,
		orders:       make(map[string]interface{}),
	}
}

// Receive handles incoming messages
func (o *OrderManagerActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		o.onStarted(ctx)
	case actor.Stopped:
		o.onStopped(ctx)
	case StatusMsg:
		o.onStatus(ctx)
	default:
		o.logger.Debug().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received message")
	}
}

func (o *OrderManagerActor) onStarted(ctx *actor.Context) {
	o.logger.Info().
		Str("exchange", o.exchangeName).
		Msg("Order manager actor started")
}

func (o *OrderManagerActor) onStopped(ctx *actor.Context) {
	o.logger.Info().
		Str("exchange", o.exchangeName).
		Msg("Order manager actor stopped")
}

func (o *OrderManagerActor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"exchange":    o.exchangeName,
		"active_orders": len(o.orders),
		"timestamp":   time.Now(),
	}
	
	ctx.Respond(status)
}