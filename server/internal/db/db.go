package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Device struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	MACAddress   string    `json:"mac_address"`
	IPAddress    string    `json:"ip_address"`
	NetInterface string    `json:"net_interface"`
	Icon         string    `json:"icon"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type DB struct {
	conn *sql.DB
}

func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrent reads
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}

	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS devices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		mac_address TEXT NOT NULL,
		ip_address TEXT NOT NULL DEFAULT '',
		net_interface TEXT NOT NULL DEFAULT 'br-lan',
		icon TEXT NOT NULL DEFAULT 'desktop',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := d.conn.Exec(query)
	return err
}

func (d *DB) ListDevices() ([]Device, error) {
	rows, err := d.conn.Query("SELECT id, name, mac_address, ip_address, net_interface, icon, created_at, updated_at FROM devices ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var dev Device
		if err := rows.Scan(&dev.ID, &dev.Name, &dev.MACAddress, &dev.IPAddress, &dev.NetInterface, &dev.Icon, &dev.CreatedAt, &dev.UpdatedAt); err != nil {
			return nil, err
		}
		devices = append(devices, dev)
	}

	if devices == nil {
		devices = []Device{}
	}

	return devices, rows.Err()
}

func (d *DB) GetDevice(id int64) (*Device, error) {
	var dev Device
	err := d.conn.QueryRow(
		"SELECT id, name, mac_address, ip_address, net_interface, icon, created_at, updated_at FROM devices WHERE id = ?", id,
	).Scan(&dev.ID, &dev.Name, &dev.MACAddress, &dev.IPAddress, &dev.NetInterface, &dev.Icon, &dev.CreatedAt, &dev.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &dev, nil
}

func (d *DB) CreateDevice(dev *Device) error {
	result, err := d.conn.Exec(
		"INSERT INTO devices (name, mac_address, ip_address, net_interface, icon) VALUES (?, ?, ?, ?, ?)",
		dev.Name, dev.MACAddress, dev.IPAddress, dev.NetInterface, dev.Icon,
	)
	if err != nil {
		return err
	}
	dev.ID, err = result.LastInsertId()
	return err
}

func (d *DB) UpdateDevice(dev *Device) error {
	_, err := d.conn.Exec(
		"UPDATE devices SET name = ?, mac_address = ?, ip_address = ?, net_interface = ?, icon = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		dev.Name, dev.MACAddress, dev.IPAddress, dev.NetInterface, dev.Icon, dev.ID,
	)
	return err
}

func (d *DB) DeleteDevice(id int64) error {
	result, err := d.conn.Exec("DELETE FROM devices WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
