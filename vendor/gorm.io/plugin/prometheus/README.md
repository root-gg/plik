# Prometheus

Collect DB Status with Prometheus

## Usage

```go
import (
  "gorm.io/gorm"
  "gorm.io/driver/sqlite"
  "gorm.io/plugin/prometheus"
)

db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})

db.Use(prometheus.New(prometheus.Config{
    DBName:          "db1", // `DBName` as metrics label
    RefreshInterval: 15,    // refresh metrics interval (default 15 seconds)
    PushAddr:        "prometheus pusher address", // push metrics if `PushAddr` configured
    StartServer:     true,  // start http server to expose metrics
    HTTPServerPort:  8080,  // configure http server port, default port 8080 (if you have configured multiple instances, only the first `HTTPServerPort` will be used to start server)
    MetricsCollector: []prometheus.MetricsCollector {
      &prometheus.MySQL{VariableNames: []string{"Threads_running"}},
 },
}))
```
