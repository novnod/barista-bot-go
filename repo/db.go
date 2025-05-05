package repo

import (
	"database/sql"
	"fmt"

	_ "github.com/glebarez/sqlite"
)

// DailyConfig holds the daily posting settings for a guild
type DailyConfig struct {
	GuildID   string
	ChannelID string
	TimeHHMM  string
}

// DailyRepository wraps a SQL DB for daily configs
type DailyRepository struct {
	db *sql.DB
}

// InitDBConnection opens (or creates) the SQLite database at dbPath
// and applies the necessary schema for daily_config.
func InitDBConnection(dbPath string) (*sql.DB, error) {
	// Use the "sqlite" driver and enable foreign keys
	dsn := fmt.Sprintf("file:%s?cache=shared&_foreign_keys=1", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	// single connection is fine for this bot
	db.SetMaxOpenConns(1)

	// apply schema
	schema := `
CREATE TABLE IF NOT EXISTS daily_config (
    guild_id   TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    time_hhmm  TEXT NOT NULL
);
`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}
	return db, nil
}

// InitDailyRepository returns a new repository bound to db
func InitDailyRepository(db *sql.DB) *DailyRepository {
	return &DailyRepository{db: db}
}

// SetConfig inserts or updates the daily config for a guild
func (r *DailyRepository) SetConfig(guildID, channelID, timeHHMM string) error {
	_, err := r.db.Exec(
		`INSERT INTO daily_config(guild_id, channel_id, time_hhmm)
         VALUES(?, ?, ?)
         ON CONFLICT(guild_id) DO UPDATE SET
             channel_id=excluded.channel_id,
             time_hhmm=excluded.time_hhmm;`,
		guildID, channelID, timeHHMM,
	)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}
	return nil
}

// GetConfig retrieves the daily config for a guild
func (r *DailyRepository) GetConfig(guildID string) (*DailyConfig, error) {
	row := r.db.QueryRow(
		`SELECT guild_id, channel_id, time_hhmm FROM daily_config WHERE guild_id = ?`,
		guildID,
	)
	var cfg DailyConfig
	if err := row.Scan(&cfg.GuildID, &cfg.ChannelID, &cfg.TimeHHMM); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return &cfg, nil
}
