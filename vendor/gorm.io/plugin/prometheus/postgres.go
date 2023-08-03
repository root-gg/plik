package prometheus

import (
	"context"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Posgres metrics providers. Metrics is being contructed from Sturct labels:
// Type translation:
//					 int64 - Counter, Gauge
// 					 time.Time - Gauge
//					 string - Label on the metrics
// Example:
//
// DatName   string `gorm:"column:datname" type:"label" help:"Name of current database"`
// SizeBytes int64  `gorm:"column:size_bytes" type:"gauge" help:"Size of database in bytes"`
//
// will be translated to:
// gorm_dbstats_size_bytes{datname="test"} 123456789
type Postgres struct {
	Prefix        string
	Interval      uint32
	VariableNames []string
	gauges        map[string]prometheus.Gauge
	counters      map[string]prometheus.Counter
	lock          sync.RWMutex
}

func (m *Postgres) getGauge(identifier string) (prometheus.Gauge, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	g, ok := m.gauges[identifier]
	return g, ok
}

func (m *Postgres) setGauge(identifier string, g prometheus.Gauge) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.gauges[identifier] = g
}

func (m *Postgres) getCounter(identifier string) (prometheus.Counter, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	c, ok := m.counters[identifier]
	return c, ok
}

func (m *Postgres) setCounter(identifier string, c prometheus.Counter) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.counters[identifier] = c
}

func (m *Postgres) Metrics(p *Prometheus) []prometheus.Collector {
	if m.Prefix == "" {
		m.Prefix = "gorm_status_"
	}

	if m.Interval == 0 {
		m.Interval = p.RefreshInterval
	}

	if m.gauges == nil {
		m.gauges = map[string]prometheus.Gauge{}
	}

	if m.counters == nil {
		m.counters = map[string]prometheus.Counter{}
	}

	funM := []func(*Prometheus, *sync.WaitGroup){
		m.replicationLag,
		m.postMasterStart,
		m.pgStatUserTables,
		m.pgStatIOUserTables,
		m.size,
		m.recordCount,
	}

	go func() {
		for range time.Tick(time.Duration(m.Interval) * time.Second) {
			var wg sync.WaitGroup
			for _, f := range funM {
				wg.Add(1)
				go f(p, &wg)
			}
			wg.Wait()
		}
	}()

	var wg sync.WaitGroup
	for _, f := range funM {
		wg.Add(1)
		go f(p, &wg)
	}
	wg.Wait()

	collectors := make([]prometheus.Collector, 0, len(m.gauges)+len(m.counters))

	for _, v := range m.gauges {
		collectors = append(collectors, v)
	}
	for _, v := range m.counters {
		collectors = append(collectors, v)
	}

	return collectors
}

func (m *Postgres) replicationLag(p *Prometheus, wg *sync.WaitGroup) {
	defer wg.Done()

	metric := "lag"

	rows, err := p.DB.Raw("SELECT CASE WHEN NOT pg_is_in_recovery() THEN 0 ELSE GREATEST (0, EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp()))) END AS lag").Rows()

	if err != nil {
		p.DB.Logger.Error(context.Background(), "gorm:prometheus query error: %v", err)
		return
	}

	var variableValue string
	for rows.Next() {
		err = rows.Scan(&variableValue)
		if err != nil {
			p.DB.Logger.Error(context.Background(), "gorm:prometheus scan got error: %v", err)
			continue
		}

		value, err := strconv.ParseFloat(variableValue, 64)
		if err != nil {
			p.DB.Logger.Error(context.Background(), "gorm:prometheus parse float got error: %v", err)
			continue
		}

		gauge, ok := m.getGauge(metric)
		if !ok {
			gauge = prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        m.Prefix + metric,
				ConstLabels: p.Labels,
				Help:        "Replication lag behind master in seconds",
			})

			m.setGauge(metric, gauge)
			prometheus.Register(gauge)
		}
		gauge.Set(value)
	}
}

func (m *Postgres) postMasterStart(p *Prometheus, wg *sync.WaitGroup) {
	defer wg.Done()

	metric := "start_time_seconds"
	rows, err := p.DB.Raw("SELECT pg_postmaster_start_time as start_time_seconds from pg_postmaster_start_time()").Rows()

	if err != nil {
		p.DB.Logger.Error(context.Background(), "gorm:prometheus query error: %v", err)
		return
	}

	var variableValue string
	for rows.Next() {
		err = rows.Scan(&variableValue)
		if err != nil {
			p.DB.Logger.Error(context.Background(), "gorm:prometheus scan got error: %v", err)
			continue
		}

		value, err := time.Parse(time.RFC3339, variableValue)
		if err != nil {
			p.DB.Logger.Error(context.Background(), "gorm:prometheus parse float got error: %v", err)
			continue
		}

		gauge, ok := m.getGauge(metric)
		if !ok {
			gauge = prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        m.Prefix + metric,
				ConstLabels: p.Labels,
				Help:        "Time unix timestamp at which postmaster started",
			})

			m.setGauge(metric, gauge)
			prometheus.Register(gauge)
		}

		gauge.Set(float64(value.Unix()))
	}
}

func (m *Postgres) size(p *Prometheus, wg *sync.WaitGroup) {
	defer wg.Done()

	type data struct {
		DatName   string `gorm:"column:datname" type:"label" help:"Name of current database"`
		SizeBytes int64  `gorm:"column:size_bytes" type:"gauge" help:"Size of database in bytes"`
	}

	rows, err := p.DB.Raw("SELECT pg_database.datname, pg_database_size(pg_database.datname) as size_bytes FROM pg_database").Rows()

	if err != nil {
		p.DB.Logger.Error(context.Background(), "gorm:prometheus query error: %v", err)
		return
	}

	for rows.Next() {
		var r data
		err = p.DB.ScanRows(rows, &r)
		if err != nil {
			p.DB.Logger.Error(context.Background(), "gorm:prometheus scan got error: %v", err)
			continue
		}

		t := reflect.TypeOf(r)
		v := reflect.ValueOf(r)

		m._parse(t, v, p)
	}
}

func (m *Postgres) pgStatUserTables(p *Prometheus, wg *sync.WaitGroup) {
	defer wg.Done()

	type data struct {
		DatName              string    `gorm:"column:datname" type:"label" help:"Name of current database"`
		SchemaName           string    `gorm:"column:schemaname" type:"label" help:"Name of the schema that this table is in"`
		Relname              string    `gorm:"column:relname" type:"label" help:"Name of this table"`
		SeqScan              int64     `gorm:"column:seq_scan" type:"counter" help:"Number of sequential scans initiated on this table"`
		SeqTupRead           int64     `gorm:"column:seq_tup_read" type:"counter" help:"Number of live rows fetched by sequential scans"`
		IdxScan              int64     `gorm:"column:idx_scan" type:"counter" help:"Number of index scans initiated on this table"`
		IdxTupFetch          int64     `gorm:"column:idx_tup_fetch" type:"counter" help:"Number of live rows fetched by index scans"`
		NTupIns              int64     `gorm:"column:n_tup_ins" type:"counter" help:"Number of rows inserted"`
		NTupUpd              int64     `gorm:"column:n_tup_upd" type:"counter" help:"Number of rows updated"`
		NTupDel              int64     `gorm:"column:n_tup_del" type:"counter" help:"Number of rows deleted"`
		NTupHotUpd           int64     `gorm:"column:n_tup_hot_upd" type:"counter" help:"Number of rows HOT updated (i.e., with no separate index update required)"`
		NLiveTup             int64     `gorm:"column:n_live_tup" type:"gauge" help:"Estimated number of live rows"`
		NDeadTup             int64     `gorm:"column:n_dead_tup" type:"gauge" help:"Estimated number of dead rows"`
		NModSinceLastAnalyze int64     `gorm:"column:n_mod_since_last_analyze" type:"gauge" help:"Estimated number of rows changed since last analyze"`
		LastVacum            time.Time `gorm:"column:last_vacuum" type:"gauge" help:"Last time at which this table was manually vacuumed (not counting VACUUM FULL)"`
		LastAutovacum        time.Time `gorm:"column:last_autovacuum" type:"gauge" help:"Last time at which this table was vacuumed by the autovacuum daemon"`
		LastAnalyze          time.Time `gorm:"column:last_analyze" type:"gauge" help:"Last time at which this table was manually analyzed"`
		LatAutoAnalyze       time.Time `gorm:"column:last_autoanalyze" type:"gauge" help:"Last time at which this table was analyzed by the autovacuum daemon"`
		VacumCount           int64     `gorm:"column:vacuum_count" type:"counter" help:"Number of times this table has been manually vacuumed (not counting VACUUM FULL)"`
		AutoVacuumCount      int64     `gorm:"column:autovacuum_count" type:"counter" help:"Number of times this table has been vacuumed by the autovacuum daemon"`
		AnalyzeCount         int64     `gorm:"column:analyze_count" type:"counter" help:"Number of times this table has been manually analyzed"`
		AutoAnalyzeCount     int64     `gorm:"column:autoanalyze_count" type:"counter" help:"Number of times this table has been analyzed by the autovacuum daemon"`
	}

	rows, err := p.DB.Raw(`
  SELECT
	current_database() datname,
	schemaname,
	relname,
	seq_scan,
	seq_tup_read,
	idx_scan,
	idx_tup_fetch,
	n_tup_ins,
	n_tup_upd,
	n_tup_del,
	n_tup_hot_upd,
	n_live_tup,
	n_dead_tup,
	n_mod_since_analyze,
	COALESCE(last_vacuum, '1970-01-01Z') as last_vacuum,
	COALESCE(last_autovacuum, '1970-01-01Z') as last_autovacuum,
	COALESCE(last_analyze, '1970-01-01Z') as last_analyze,
	COALESCE(last_autoanalyze, '1970-01-01Z') as last_autoanalyze,
	vacuum_count,
	autovacuum_count,
	analyze_count,
	autoanalyze_count
  FROM
	pg_stat_user_tables`).Rows()

	if err != nil {
		p.DB.Logger.Error(context.Background(), "gorm:prometheus query error: %v", err)
		return
	}

	for rows.Next() {
		var r data
		err = p.DB.ScanRows(rows, &r)
		if err != nil {
			p.DB.Logger.Error(context.Background(), "gorm:prometheus scan got error: %v", err)
			continue
		}

		t := reflect.TypeOf(r)
		v := reflect.ValueOf(r)

		m._parse(t, v, p)
	}
}

func (m *Postgres) pgStatIOUserTables(p *Prometheus, wg *sync.WaitGroup) {
	defer wg.Done()

	type data struct {
		DatName       string `gorm:"column:datname" type:"label" help:"Name of current database"`
		SchemaName    string `gorm:"column:schemaname" type:"label" help:"Name of the schema that this table is in"`
		Relname       string `gorm:"column:relname" type:"label" help:"Name of this table"`
		HeapBlksRead  int64  `gorm:"column:heap_blks_read" type:"counter" help:"Number of disk blocks read from this table"`
		HeapBlksHit   int64  `gorm:"column:heap_blks_hit" type:"counter" help:"Number of buffer hits in this table"`
		IdxBlksRead   int64  `gorm:"column:idx_blks_read" type:"counter" help:"Number of disk blocks read from all indexes on this table"`
		IdxBlksHit    int64  `gorm:"column:idx_blks_hit" type:"counter" help:"Number of buffer hits in all indexes on this table"`
		ToastBlksRead int64  `gorm:"column:toast_blks_read" type:"counter" help:"Number of disk blocks read from this table's TOAST table (if any)"`
		ToastBlksHit  int64  `gorm:"column:toast_blks_hit" type:"counter" help:"Number of buffer hits in this table's TOAST table (if any)"`
		TidxBlksRead  int64  `gorm:"column:toast_idx_blks_read" type:"counter" help:"Number of disk blocks read from this table's TOAST table indexes (if any)"`
		TidxBlksHit   int64  `gorm:"column:toast_idx_blks_hit" type:"counter" help:"Number of buffer hits in this table's TOAST table indexes (if any)"`
	}

	rows, err := p.DB.Raw(`SELECT 
	 current_database() datname,
	 schemaname, 
	 relname, 
	 heap_blks_read, 
	 heap_blks_hit, 
	 idx_blks_read, 
	 idx_blks_hit, 
	 toast_blks_read, 
	 toast_blks_hit, 
	 tidx_blks_read, 
	 tidx_blks_hit 
    FROM 
	 pg_statio_user_tables`).Rows()

	if err != nil {
		p.DB.Logger.Error(context.Background(), "gorm:prometheus query error: %v", err)
		return
	}

	for rows.Next() {
		var r data
		err = p.DB.ScanRows(rows, &r)
		if err != nil {
			p.DB.Logger.Error(context.Background(), "gorm:prometheus scan got error: %v", err)
			continue
		}

		t := reflect.TypeOf(r)
		v := reflect.ValueOf(r)

		m._parse(t, v, p)
	}
}

func (m *Postgres) recordCount(p *Prometheus, wg *sync.WaitGroup) {
	defer wg.Done()

	type data struct {
		DatName    string `gorm:"column:table_schema" type:"label" help:"Name of current database"`
		SchemaName string `gorm:"column:table_name" type:"label" help:"Name of the schema that this table is in"`
		RowsCount  int64  `gorm:"column:rows_count" type:"gauge" help:"Name of this table"`
	}

	rows, err := p.DB.Raw(`with tbl as (SELECT table_schema,table_name FROM information_schema.tables   where table_name not like 'pg_%' and table_schema in ('public'))   select table_schema, table_name, (xpath('/row/c/text()', query_to_xml(format('select count(*) as c from %I.%I', table_schema, table_name), false, true, '')))[1]::text::int as rows_count from tbl ORDER BY 3 DESC;`).Rows()

	if err != nil {
		p.DB.Logger.Error(context.Background(), "gorm:prometheus query error: %v", err)
		return
	}

	for rows.Next() {
		var r data
		err = p.DB.ScanRows(rows, &r)
		if err != nil {
			p.DB.Logger.Error(context.Background(), "gorm:prometheus scan got error: %v", err)
			continue
		}

		t := reflect.TypeOf(r)
		v := reflect.ValueOf(r)

		m._parse(t, v, p)
	}
}

// _parse parses the data per database ROW and registers it for sending
func (m *Postgres) _parse(t reflect.Type, v reflect.Value, p *Prometheus) {
	labels := map[string]string{}

	// sort out labels for all row. This helps to identify unique metric
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("type")
		if tag == "label" {
			labels[strings.TrimPrefix(field.Tag.Get("gorm"), "column:")] = v.Field(i).String()
		}
	}

	// emit the metrics
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("type")

		// identifier for status store
		identifier := strings.TrimPrefix(field.Tag.Get("gorm"), "column:")
		// metric itself
		metric := identifier

		for _, l := range labels {
			identifier = identifier + "_" + l
		}

		// we register required metrics first
		switch tag {
		case "label":
			// do nothing
		case "gauge":
			// field name + labels are unique status identifier
			_, ok := m.getGauge(identifier)
			if !ok {
				for k, v := range p.Labels {
					labels[k] = v
				}
				g := prometheus.NewGauge(prometheus.GaugeOpts{
					Name:        m.Prefix + metric,
					ConstLabels: labels,
					Help:        field.Tag.Get("help"),
				})

				m.setGauge(identifier, g)
				prometheus.Register(g)
			}
		case "counter":
			_, ok := m.getCounter(identifier)
			if !ok {
				for k, v := range p.Labels {
					labels[k] = v
				}
				c := prometheus.NewCounter(prometheus.CounterOpts{
					Name:        m.Prefix + metric,
					ConstLabels: labels,
					Help:        field.Tag.Get("help"),
				})

				m.setCounter(identifier, c)
				prometheus.Register(c)
			}
		default:
			p.DB.Logger.Error(context.Background(), "gorm:prometheus unhandled type: %s", tag)
			continue
		}

		switch v.Field(i).Interface().(type) {
		case string:
		case int64:
			value := v.Field(i).Interface().(int64)
			switch tag {
			case "gauge":
				g, ok := m.getGauge(identifier)
				if ok {
					g.Set(float64(value))
					m.setGauge(identifier, g)
				}
			case "counter":
				c, ok := m.getCounter(identifier)
				if ok {
					c.Add(float64(value))
					m.setCounter(identifier, c)
				}
			default:
				p.DB.Logger.Error(context.Background(), "gorm:prometheus unhandled type: %s %v", tag, v.Field(i).Type())
				continue
			}
		case time.Time:
			switch tag {
			case "gauge":
				value, err := time.Parse(time.RFC3339, v.Field(i).Interface().(time.Time).Format(time.RFC3339))
				if err != nil {
					p.DB.Logger.Error(context.Background(), "gorm:prometheus parse float got error: %v", err)
					continue
				}
				g, ok := m.getGauge(identifier)
				if ok {
					g.Set(float64(value.Unix()))
					m.setGauge(identifier, g)
				}
			default:
				p.DB.Logger.Error(context.Background(), "gorm:prometheus unhandled type: %s %v", tag, v.Field(i).Type())
				continue
			}
		default:
			p.DB.Logger.Error(context.Background(), "gorm:prometheus unhandled type: %v", v.Field(i).Type())
			continue
		}
	}
}
