## 🔍 Nmap-API: Distributed Port Scanning and Change Tracking Tool

A containerized, scalable REST API built in Golang that uses `nmap` under the hood to scan IP addresses or hostnames, persist results, detect changes between scans, and expose metrics via OpenTelemetry.

This tool is designed to help monitor and scans for ports over time — in both real-time and historical contexts.

---

### 🚀 Key Features

- REST API to initiate port scans on multiple hosts in parallel
- Historical scan results stored in PostgreSQL
- Diff engine to highlight newly opened or closed ports
- Background job queue powered by Redis
- Prometheus-compatible metrics (OTLP via OpenTelemetry)
- End-to-end tracing with Jaeger
- Auto-recovery on failure with detailed logging
- Runs fully in Docker via Compose

---

### Flow Diagram

![Architecture Diagram](/docs/flow_diagram.png)
---

### 🧭 Supported Endpoints

<img src="/docs/swagger.png" alt="Architecture Diagram" style="height: 70%;">

#### 1. **Initiate Scan**
```http
POST /scan
```
**Description:** Launches background port scans for one or more hosts.

**Input:**
```json
{
  "hosts": ["scanme.nmap.org", "example.com"]
}
```

**Output:**
```json
{
  "scan_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

---

#### 2. **Fetch Results**
```http
GET /results/:host
```
Returns up to 10 recent scan results for a specific host.

**Example Response:**
```json
[
  {
    "scan_id": "abc-123",
    "host": "scanme.nmap.org",
    "scanned_at": "2025-05-09T07:00:00Z",
    "open_ports": [22, 80, 443]
  }
]
```

---

#### 3. **Fetch Scan Diffs**
```http
GET /diff/:host
```
Compares last two scans and shows newly opened/closed ports.

**Output:**
```json
{
  "host": "scanme.nmap.org",
  "newly_opened": [8080],
  "newly_closed": [3306]
}
```

---

#### 4. **Poll Scan Status**
```http
GET /scan/status/:scan_id
```

Returns the status of all hosts under a scan ID.

**Output:**
```json
[
  {
    "host": "example.com",
    "status": "done"
  },
  {
    "host": "api.dev",
    "status": "in_progress"
  }
]
```

---

### 📈 Metrics, Tracing & Reliability

This tool integrates with the **OpenTelemetry** stack:

- **Metrics Exported via OTLP**:  
  - `nmap_scans_total`: count of scans run  
  - `nmap_scan_failures_total`: failed scans  
  - `nmap_scan_duration_seconds`: duration histogram
      ![Prometheus](/docs/prometheus.png)

- **Distributed Tracing**:
  - Trace every step: queueing, scan execution, DB insert, Redis I/O
  - Integrated with Jaeger UI at [`localhost:16686`](http://localhost:16686)
      ![Jaeger](/docs/jaeger.png)

- **System Uptime Handling**:
  - Background worker with Redis BLPOP ensures reliable queueing
  - Scan status table ensures progress is tracked and is recoverable at any point in time
  - Docker Compose handles restart policies and isolation


---

### ⚠️ Limitations and Areas of Improvement

While Nmap is powerful, it's not the fastest for internet-wide scanning. These tools can be integrated or used as alternatives:

- [`masscan`](https://github.com/robertdavidgraham/masscan): Ultra-fast port scanner
- [`rustscan`](https://github.com/RustScan/RustScan): Leverages Nmap but runs in parallel faster
- Using an adaptive strategy (Nmap fallback if other scans fail) is ideal

---

### ⚙️ Scaling to 1M RPS: Design Considerations

To scale this system to handle 1 million requests per second:

| Component | Upgrade Strategy |
|----------|------------------|
| **API Layer & Autoscaling** | Run behind a load balancer with autoscaled replicas /on demand replicas |
| **Queueing** | Switch Redis to a distributed message broker like Kafka or using redis clusters |
| **Scanning** | Use container pools or lambda-style scan workers with `rustscan` |
| **Storage** | Partition scan results by time or host; consider using ClickHouse for analytics |
| **Metrics/Trace** | Offload to dedicated OpenTelemetry Collector nodes |
| **Trade-Offs** | Need to balance scan thoroughness vs. speed; possible false negatives with shallow scanning |

---

### 🐳 Deployment

```bash
docker compose up --build
```

Access services:
- API: `http://localhost:8080`
- Swagger: `http://localhost:8080/swagger/index.html`
- Prometheus: `http://localhost:9090`
- Jaeger: `http://localhost:16686`
- Redis: `localhost:6379`
- PostgreSQL: `localhost:5432`

---

### 🙌 Contributing

Feel free to fork and extend this project. Ideal contributions:
- Swap Nmap for RustScan backend
- Add scan scheduling and notification hooks
- Scale scan workers horizontally via queue coordination

---

