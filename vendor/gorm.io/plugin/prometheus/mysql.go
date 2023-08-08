package prometheus

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type MySQL struct {
	Prefix        string
	Interval      uint32
	VariableNames []string
	status        map[string]prometheus.Gauge
}

func (m *MySQL) Metrics(p *Prometheus) []prometheus.Collector {
	if m.Prefix == "" {
		m.Prefix = "gorm_status_"
	}

	if m.Interval == 0 {
		m.Interval = p.RefreshInterval
	}

	if m.status == nil {
		m.status = map[string]prometheus.Gauge{}
	}

	go func() {
		for range time.Tick(time.Duration(m.Interval) * time.Second) {
			m.collect(p)
		}
	}()

	m.collect(p)
	collectors := make([]prometheus.Collector, 0, len(m.status))

	for _, v := range m.status {
		collectors = append(collectors, v)
	}

	return collectors
}

func (m *MySQL) collect(p *Prometheus) {
	rows, err := p.DB.Raw("SHOW STATUS").Rows()

	if err != nil {
		p.DB.Logger.Error(context.Background(), "gorm:prometheus query error: %v", err)
		return
	}

	var variableName, variableValue string
	for rows.Next() {
		err = rows.Scan(&variableName, &variableValue)
		if err != nil {
			p.DB.Logger.Error(context.Background(), "gorm:prometheus scan got error: %v", err)
			continue
		}

		var found = len(m.VariableNames) == 0

		for _, name := range m.VariableNames {
			if name == variableName {
				found = true
				break
			}
		}

		if found {
			value, err := strconv.ParseFloat(variableValue, 64)
			if err != nil {
				p.DB.Logger.Error(context.Background(), "gorm:prometheus parse float got error: %v", err)
				continue
			}

			gauge, ok := m.status[variableName]
			if !ok {
				gauge = prometheus.NewGauge(prometheus.GaugeOpts{
					Name:        m.Prefix + variableName,
					ConstLabels: p.Labels,
				})

				m.status[variableName] = gauge
				_ = prometheus.Register(gauge)
			}

			gauge.Set(value)
		}
	}

	return
}
