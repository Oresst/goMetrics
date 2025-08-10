package store

import (
	"errors"
	"github.com/Oresst/goMetrics/models"
	"github.com/jackc/pgx"
	log "github.com/sirupsen/logrus"
)

type DBStorage struct {
	db *pgx.Conn
}

func NewDBStorage(db *pgx.Conn) *DBStorage {
	return &DBStorage{db: db}
}

func (s *DBStorage) AddMetric(metricType string, name string, value float64) error {
	var sql string
	var err error
	var metric models.Metrics

	row := s.db.QueryRow("SELECT name, type, delta, value FROM metrics WHERE name = $1 AND type = $2", name, metricType)
	err = row.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		if metricType == models.Counter {
			sql = "INSERT INTO metrics (type, name, delta) VALUES ($1, $2, $3)"
			_, err = s.db.Exec(sql, metricType, name, int64(value))
		} else {
			sql = "INSERT INTO metrics (type, name, value) VALUES ($1, $2, $3)"
			_, err = s.db.Exec(sql, metricType, name, value)
		}
	} else {
		if metricType == models.Counter {
			sql = "UPDATE metrics SET delta = $1 WHERE name = $2 AND type = $3"
			_, err = s.db.Exec(sql, int64(value)+*metric.Delta, metric.ID, metric.MType)
		} else {
			sql = "UPDATE metrics SET value = $1 WHERE name = $2 AND type = $3"
			_, err = s.db.Exec(sql, value, metric.ID, metric.MType)
		}
	}

	if err != nil {
		return err
	}

	return nil
}

func (s *DBStorage) GetMetric(name string) (float64, error) {
	sql := "SELECT type, name, delta, value FROM metrics WHERE name = $1 ORDER BY id DESC LIMIT 1"
	var metric models.Metrics

	row := s.db.QueryRow(sql, name)
	err := row.Scan(&metric.MType, &metric.ID, &metric.Delta, &metric.Value)
	if err != nil {
		if errors.Is(pgx.ErrNoRows, err) {
			return 0, errors.New("metric not found")
		}

		return 0, err
	}

	if metric.MType == models.Counter {
		return float64(*metric.Delta), nil
	} else {
		return *metric.Value, nil
	}
}

func (s *DBStorage) GetAllMetrics() map[string]models.Metrics {
	metrics := make(map[string]models.Metrics)

	sql := `
			SELECT type, name, delta, value
			FROM metrics
		`

	rows, err := s.db.Query(sql)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Error getting metrics")
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var metric models.Metrics
		err = rows.Scan(&metric.MType, &metric.ID, &metric.Delta, &metric.Value)

		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Error getting metrics")
			return nil
		}

		metrics[metric.ID] = metric
	}

	return metrics
}
