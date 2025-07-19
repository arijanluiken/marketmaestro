package database

import (
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Order represents an order in the database
type Order struct {
	ID              int64  // Database ID (auto-increment)
	ExchangeOrderID string // Order ID from exchange
	Exchange        string
	Symbol          string
	Side            string
	Type            string
	Quantity        float64
	Price           float64
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// DB represents the database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection and runs migrations
func New(dbPath string) (*DB, error) {
	// Open database connection
	conn, err := sql.Open("sqlite3", dbPath+"?_fk=1&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}

	// Run migrations
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// migrate runs database migrations
func (db *DB) migrate() error {
	driver, err := sqlite3.WithInstance(db.conn, &sqlite3.Config{})
	if err != nil {
		return err
	}

	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// SaveOrder saves an order to the database
func (db *DB) SaveOrder(order *Order) error {
	query := `
		INSERT INTO orders (order_id, exchange, symbol, side, type, quantity, price, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(exchange, order_id) DO UPDATE SET
			status = excluded.status,
			price = excluded.price,
			updated_at = excluded.updated_at
	`

	result, err := db.conn.Exec(query,
		order.ExchangeOrderID,
		order.Exchange,
		order.Symbol,
		order.Side,
		order.Type,
		order.Quantity,
		order.Price,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
	)

	if err != nil {
		return err
	}

	// Set the ID for new inserts
	if order.ID == 0 {
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		order.ID = id
	}

	return nil
}

// GetAllOpenOrders retrieves all open orders from the database
func (db *DB) GetAllOpenOrders() ([]*Order, error) {
	query := `
		SELECT id, order_id, exchange, symbol, side, type, quantity, price, status, created_at, updated_at
		FROM orders 
		WHERE status IN ('open', 'partially_filled', 'pending')
		ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*Order
	for rows.Next() {
		order := &Order{}
		err := rows.Scan(
			&order.ID,
			&order.ExchangeOrderID,
			&order.Exchange,
			&order.Symbol,
			&order.Side,
			&order.Type,
			&order.Quantity,
			&order.Price,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// UpdateOrderStatus updates the status of an order by exchange order ID
func (db *DB) UpdateOrderStatus(exchangeOrderID, status string) error {
	query := `UPDATE orders SET status = ?, updated_at = ? WHERE order_id = ?`
	_, err := db.conn.Exec(query, status, time.Now(), exchangeOrderID)
	return err
}

// UpdateOrder updates an order in the database
func (db *DB) UpdateOrder(order *Order) error {
	query := `
		UPDATE orders 
		SET status = ?, price = ?, updated_at = ?
		WHERE order_id = ? AND exchange = ?
	`
	_, err := db.conn.Exec(query,
		order.Status,
		order.Price,
		order.UpdatedAt,
		order.ExchangeOrderID,
		order.Exchange,
	)
	return err
}

// Conn returns the underlying database connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}
