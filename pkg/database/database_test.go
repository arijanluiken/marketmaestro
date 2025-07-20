package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("creates database and runs migrations", func(t *testing.T) {
		// Create temporary database file
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := New(dbPath)
		if err != nil {
			t.Fatalf("expected no error creating database, got %v", err)
		}
		defer db.Close()

		// Verify the database file was created
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("expected database file to be created")
		}

		// Verify tables exist by querying them
		tables := []string{"settings", "orders", "positions", "portfolio_snapshots", "strategy_runs"}
		for _, table := range tables {
			query := "SELECT COUNT(*) FROM " + table
			var count int
			err := db.conn.QueryRow(query).Scan(&count)
			if err != nil {
				t.Errorf("expected table %s to exist, got error: %v", table, err)
			}
		}
	})

	t.Run("fails with invalid path", func(t *testing.T) {
		// Try to create database in non-existent directory
		invalidPath := "/nonexistent/directory/test.db"
		
		_, err := New(invalidPath)
		if err == nil {
			t.Error("expected error for invalid path, got nil")
		}
	})
}

func TestSaveOrder(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	t.Run("saves new order", func(t *testing.T) {
		order := &Order{
			ExchangeOrderID: "order123",
			Exchange:        "bybit",
			Symbol:          "BTCUSDT",
			Side:            "buy",
			Type:            "limit",
			Quantity:        1.0,
			Price:           50000.0,
			Status:          "open",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		err := db.SaveOrder(order)
		if err != nil {
			t.Fatalf("expected no error saving order, got %v", err)
		}

		// Verify the order ID was set
		if order.ID == 0 {
			t.Error("expected order ID to be set after save")
		}

		// Verify the order was saved
		var count int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM orders WHERE order_id = ?", order.ExchangeOrderID).Scan(&count)
		if err != nil {
			t.Fatalf("error querying saved order: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 order, found %d", count)
		}
	})

	t.Run("updates existing order on conflict", func(t *testing.T) {
		order := &Order{
			ExchangeOrderID: "order456",
			Exchange:        "bybit",
			Symbol:          "ETHUSDT",
			Side:            "sell",
			Type:            "limit",
			Quantity:        2.0,
			Price:           3000.0,
			Status:          "open",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		// Save initial order
		err := db.SaveOrder(order)
		if err != nil {
			t.Fatalf("expected no error saving initial order, got %v", err)
		}

		// Update the order
		order.Status = "filled"
		order.Price = 3100.0
		order.UpdatedAt = time.Now().Add(time.Minute)

		err = db.SaveOrder(order)
		if err != nil {
			t.Fatalf("expected no error updating order, got %v", err)
		}

		// Verify only one order exists with updated values
		var count int
		var status string
		var price float64
		
		err = db.conn.QueryRow(
			"SELECT COUNT(*), status, price FROM orders WHERE order_id = ? AND exchange = ?",
			order.ExchangeOrderID, order.Exchange,
		).Scan(&count, &status, &price)
		if err != nil {
			t.Fatalf("error querying updated order: %v", err)
		}
		
		if count != 1 {
			t.Errorf("expected 1 order, found %d", count)
		}
		if status != "filled" {
			t.Errorf("expected status 'filled', got '%s'", status)
		}
		if price != 3100.0 {
			t.Errorf("expected price 3100.0, got %f", price)
		}
	})
}

func TestGetAllOpenOrders(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	// Insert test orders
	orders := []*Order{
		{
			ExchangeOrderID: "open1",
			Exchange:        "bybit",
			Symbol:          "BTCUSDT",
			Side:            "buy",
			Type:            "limit",
			Quantity:        1.0,
			Price:           50000.0,
			Status:          "open",
			CreatedAt:       time.Now().Add(-time.Hour),
			UpdatedAt:       time.Now().Add(-time.Hour),
		},
		{
			ExchangeOrderID: "filled1",
			Exchange:        "bybit",
			Symbol:          "ETHUSDT",
			Side:            "sell",
			Type:            "limit",
			Quantity:        2.0,
			Price:           3000.0,
			Status:          "filled",
			CreatedAt:       time.Now().Add(-30*time.Minute),
			UpdatedAt:       time.Now().Add(-30*time.Minute),
		},
		{
			ExchangeOrderID: "partial1",
			Exchange:        "bybit",
			Symbol:          "ADAUSDT",
			Side:            "buy",
			Type:            "limit",
			Quantity:        100.0,
			Price:           1.0,
			Status:          "partially_filled",
			CreatedAt:       time.Now().Add(-15*time.Minute),
			UpdatedAt:       time.Now().Add(-15*time.Minute),
		},
		{
			ExchangeOrderID: "pending1",
			Exchange:        "bybit",
			Symbol:          "DOGEUSDT",
			Side:            "buy",
			Type:            "limit",
			Quantity:        1000.0,
			Price:           0.1,
			Status:          "pending",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	for _, order := range orders {
		err := db.SaveOrder(order)
		if err != nil {
			t.Fatalf("failed to save test order: %v", err)
		}
	}

	t.Run("returns only open orders", func(t *testing.T) {
		openOrders, err := db.GetAllOpenOrders()
		if err != nil {
			t.Fatalf("expected no error getting open orders, got %v", err)
		}

		// Should return 3 orders (open, partially_filled, pending) but not filled
		expectedCount := 3
		if len(openOrders) != expectedCount {
			t.Errorf("expected %d open orders, got %d", expectedCount, len(openOrders))
		}

		// Verify the orders are sorted by created_at DESC (newest first)
		if len(openOrders) >= 2 {
			if openOrders[0].CreatedAt.Before(openOrders[1].CreatedAt) {
				t.Error("expected orders to be sorted by created_at DESC")
			}
		}

		// Verify no filled orders are returned
		for _, order := range openOrders {
			if order.Status == "filled" {
				t.Errorf("unexpected filled order in open orders: %s", order.ExchangeOrderID)
			}
		}
	})

	t.Run("returns empty slice when no open orders", func(t *testing.T) {
		// Update all orders to filled status
		_, err := db.conn.Exec("UPDATE orders SET status = 'filled'")
		if err != nil {
			t.Fatalf("failed to update orders: %v", err)
		}

		openOrders, err := db.GetAllOpenOrders()
		if err != nil {
			t.Fatalf("expected no error getting open orders, got %v", err)
		}

		if len(openOrders) != 0 {
			t.Errorf("expected 0 open orders, got %d", len(openOrders))
		}
	})
}

func TestUpdateOrderStatus(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	// Insert test order
	order := &Order{
		ExchangeOrderID: "update_test",
		Exchange:        "bybit",
		Symbol:          "BTCUSDT",
		Side:            "buy",
		Type:            "limit",
		Quantity:        1.0,
		Price:           50000.0,
		Status:          "open",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err = db.SaveOrder(order)
	if err != nil {
		t.Fatalf("failed to save test order: %v", err)
	}

	t.Run("updates order status", func(t *testing.T) {
		err := db.UpdateOrderStatus(order.ExchangeOrderID, "filled")
		if err != nil {
			t.Fatalf("expected no error updating order status, got %v", err)
		}

		// Verify the status was updated
		var status string
		err = db.conn.QueryRow(
			"SELECT status FROM orders WHERE order_id = ?",
			order.ExchangeOrderID,
		).Scan(&status)
		if err != nil {
			t.Fatalf("error querying updated order: %v", err)
		}

		if status != "filled" {
			t.Errorf("expected status 'filled', got '%s'", status)
		}
	})

	t.Run("handles non-existent order", func(t *testing.T) {
		err := db.UpdateOrderStatus("nonexistent", "cancelled")
		if err != nil {
			t.Fatalf("expected no error for non-existent order, got %v", err)
		}
	})
}

func TestUpdateOrder(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	// Insert test order
	order := &Order{
		ExchangeOrderID: "update_order_test",
		Exchange:        "bybit",
		Symbol:          "ETHUSDT",
		Side:            "sell",
		Type:            "limit",
		Quantity:        2.0,
		Price:           3000.0,
		Status:          "open",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err = db.SaveOrder(order)
	if err != nil {
		t.Fatalf("failed to save test order: %v", err)
	}

	t.Run("updates order", func(t *testing.T) {
		// Update order fields
		order.Status = "partially_filled"
		order.Price = 3100.0
		order.UpdatedAt = time.Now().Add(time.Minute)

		err := db.UpdateOrder(order)
		if err != nil {
			t.Fatalf("expected no error updating order, got %v", err)
		}

		// Verify the order was updated
		var status string
		var price float64
		err = db.conn.QueryRow(
			"SELECT status, price FROM orders WHERE order_id = ? AND exchange = ?",
			order.ExchangeOrderID, order.Exchange,
		).Scan(&status, &price)
		if err != nil {
			t.Fatalf("error querying updated order: %v", err)
		}

		if status != "partially_filled" {
			t.Errorf("expected status 'partially_filled', got '%s'", status)
		}
		if price != 3100.0 {
			t.Errorf("expected price 3100.0, got %f", price)
		}
	})

	t.Run("handles non-existent order", func(t *testing.T) {
		nonExistentOrder := &Order{
			ExchangeOrderID: "nonexistent",
			Exchange:        "bybit",
			Status:          "cancelled",
			Price:           0,
			UpdatedAt:       time.Now(),
		}

		err := db.UpdateOrder(nonExistentOrder)
		if err != nil {
			t.Fatalf("expected no error for non-existent order, got %v", err)
		}
	})
}

func TestClose(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	t.Run("closes database connection", func(t *testing.T) {
		err := db.Close()
		if err != nil {
			t.Fatalf("expected no error closing database, got %v", err)
		}

		// Verify connection is closed by trying to use it
		_, err = db.conn.Query("SELECT 1")
		if err == nil {
			t.Error("expected error using closed connection, got nil")
		}
	})
}

func TestConn(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	t.Run("returns database connection", func(t *testing.T) {
		conn := db.Conn()
		if conn == nil {
			t.Error("expected non-nil connection")
		}

		// Test that we can use the connection
		var result int
		err := conn.QueryRow("SELECT 1").Scan(&result)
		if err != nil {
			t.Errorf("expected no error using connection, got %v", err)
		}
		if result != 1 {
			t.Errorf("expected result 1, got %d", result)
		}
	})
}