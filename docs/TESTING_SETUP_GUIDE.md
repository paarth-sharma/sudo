# Complete Testing Setup Guide

> 📖 **Related Documentation:**
> - [README](README.md) - Project overview and quick start
> - [Self-Hosting Guide](SELF_HOST.md) - Production deployment and monitoring
> - [Security Guide](SECURITY.md) - Encryption and security implementation

This guide walks you through setting up and running the complete testing infrastructure for SUDO Kanban, from basic setup to advanced load testing.

**What you'll learn:**
- Unit and integration testing setup
- Load testing with Artillery
- Performance benchmarking
- Monitoring and metrics
- Test result interpretation

## 📋 Prerequisites

### Required Software
```bash
# Check if you have these installed:
go version          # Should be 1.21+
node --version      # Should be 18+
npm --version       # Should be 8+
docker --version    # Should be 20+
git --version       # Any recent version
```

### Install Missing Dependencies
```bash
# Install Node.js dependencies
npm install

# Install Go testing dependencies
go install github.com/stretchr/testify@latest

# Install Artillery for load testing
npm install -g artillery

# Install templ if not already installed
go install github.com/a-h/templ/cmd/templ@latest
```

## 🔧 Step 1: Initial Setup

### 1.1 Environment Configuration
```bash
# Copy the production environment template
cp .env.production.example .env.production

# Edit the production environment file
# Replace placeholder values with your actual configuration
```

**Required Environment Variables:**
```bash
# Edit .env.production with real values
JWT_SECRET=your-actual-jwt-secret-32-chars-minimum
SUPABASE_URL=your-actual-supabase-url
SUPABASE_ANON_KEY=your-actual-supabase-anon-key
```

### 1.2 Make Scripts Executable
```bash
# Make all scripts executable
chmod +x scripts/deploy.sh
chmod +x testing/load-testing/run-load-test.sh
chmod +x testing/run-tests.sh
```

### 1.3 Build the Application
```bash
# Generate templates
templ generate

# Build CSS
npm run build-css

# Build Go application
go build -o bin/server cmd/server/main.go
```

## 🧪 Step 2: Unit and Integration Testing

### 2.1 Run Basic Health Check
```bash
# Start the server in development mode
go run cmd/server/main.go

# In another terminal, check if server is running
curl http://localhost:8080/health
# Should return: {"status":"ok","version":"1.0.0"}
```

### 2.2 Run Complete Test Suite
```bash
# Stop the server (Ctrl+C) and run the test suite
cd testing
./run-tests.sh

# For quick tests only (skips performance tests)
./run-tests.sh --short
```

**What this does:**
- Runs all unit tests with coverage
- Runs WebSocket integration tests
- Runs real-time collaboration tests
- Checks for race conditions
- Generates coverage reports
- Creates a detailed test summary

### 2.3 Check Test Results
```bash
# View test results
ls test-results/

# Open coverage report in browser
open coverage.html  # macOS
start coverage.html  # Windows
xdg-open coverage.html  # Linux

# View detailed test summary
cat test-results/test_summary_*.md
```

## 🚀 Step 3: Load Testing

### 3.1 Prepare for Load Testing
```bash
# Start the server
go run cmd/server/main.go

# In another terminal, verify server is healthy
curl http://localhost:8080/health
```

### 3.2 Run WebSocket Load Tests
```bash
cd testing/load-testing

# Set environment variables for testing
export TEST_BOARD_ID=$(uuidgen)  # macOS/Linux
# For Windows: set TEST_BOARD_ID=some-uuid-here

# Run the load test
./run-load-test.sh
```

**What this does:**
- Tests WebSocket connections under load
- Simulates real-time collaboration
- Tests API endpoints under stress
- Measures response times and throughput
- Generates detailed performance reports

### 3.3 Manual Artillery Tests (Optional)
```bash
# Run specific Artillery tests manually
artillery run artillery-websocket-test.yml
artillery run artillery-api-test.yml

# Generate HTML reports
artillery report results.json --output report.html
```

## 📊 Step 4: Monitoring and Metrics

### 4.1 Start with Monitoring Stack
```bash
# Start the monitoring stack
docker-compose -f docker-compose.prod.yml up prometheus grafana

# Access monitoring dashboards
# Prometheus: http://localhost:9090
# Grafana: http://localhost:3000 (admin/admin)
```

### 4.2 Check Application Metrics
```bash
# Start server with metrics enabled
export ENABLE_METRICS=true
go run cmd/server/main.go

# View metrics endpoint
curl http://localhost:8080/metrics
```

## 🔍 Step 5: Verify Everything Works

### 5.1 End-to-End Test Checklist

Run through this checklist to verify everything is working:

```bash
# 1. Server starts without errors
go run cmd/server/main.go
# ✅ Should start on port 8080

# 2. Health check responds
curl http://localhost:8080/health
# ✅ Should return {"status":"ok"}

# 3. Metrics endpoint works
curl http://localhost:8080/metrics
# ✅ Should return JSON with metrics

# 4. Tests pass
cd testing && ./run-tests.sh --short
# ✅ Should show all tests passing

# 5. Load test runs
cd testing/load-testing && ./run-load-test.sh
# ✅ Should complete without major errors
```

### 5.2 Performance Benchmarks

Your system should meet these benchmarks:

| Metric | Target | How to Verify |
|--------|--------|---------------|
| Server startup | < 5 seconds | Time `go run cmd/server/main.go` |
| Health check response | < 100ms | Check load test results |
| WebSocket connection | < 500ms | Check Artillery reports |
| Test coverage | > 80% | Check `coverage.html` |
| Memory usage | < 512MB | Check metrics endpoint |

## 🐛 Troubleshooting

### Common Issues and Solutions

**1. "Permission denied" on scripts**
```bash
chmod +x testing/run-tests.sh
chmod +x testing/load-testing/run-load-test.sh
chmod +x scripts/deploy.sh
```

**2. "Artillery not found"**
```bash
npm install -g artillery
# Or use npx: npx artillery run artillery-websocket-test.yml
```

**3. "Database connection error"**
- Check your Supabase credentials in `.env`
- Ensure your database is accessible
- Verify network connectivity

**4. "WebSocket connection failed"**
- Ensure server is running on port 8080
- Check firewall settings
- Verify no other service is using port 8080

**5. "Tests failing"**
```bash
# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestWebSocketConnection ./testing/integration/
```

**6. "Load test shows high latency"**
- Check system resources (CPU, memory)
- Reduce concurrent connections in Artillery config
- Ensure database can handle the load

## 📈 Interpreting Results

### Test Coverage
- **Green (>80%)**: Excellent, ready for production
- **Yellow (60-80%)**: Good, consider adding more tests
- **Red (<60%)**: Needs more test coverage

### Load Test Results
- **Success Rate >95%**: Excellent
- **Response Time P95 <500ms**: Good performance
- **Error Rate <2%**: Acceptable
- **WebSocket connections >100**: Good scalability

### Key Metrics to Watch
1. **Response Times**: P50, P95, P99 percentiles
2. **Error Rates**: HTTP 4xx, 5xx responses
3. **WebSocket Metrics**: Connection success, message latency
4. **Resource Usage**: CPU, memory, goroutines

## 🎯 Next Steps

Once all tests pass:

1. **Review test reports** for any warnings or recommendations
2. **Optimize performance** based on load test results
3. **Deploy to staging** using the deployment scripts
4. **Set up monitoring** in your production environment
5. **Begin user recruitment** for Phase 3 testing

## 📞 Getting Help

If you encounter issues:

1. Check the log files in `test-results/` directory
2. Review the detailed error messages in test output
3. Ensure all prerequisites are properly installed
4. Verify environment variables are correctly set

The testing infrastructure is now ready for comprehensive validation of your real-time Kanban application!

---

## Related Guides

### After Testing

Once your tests pass, you're ready to deploy:

- **[Deploy to Production](SELF_HOST.md)** - Self-hosting guide with monitoring
- **[Security Hardening](SECURITY.md)** - Production security checklist
- **[Performance Monitoring](SELF_HOST.md#monitoring--observability)** - Set up Prometheus and Grafana

### Testing Best Practices

- Run tests before every commit
- Maintain >80% code coverage
- Test WebSocket connections under load
- Monitor performance metrics in production
- Set up automated testing in CI/CD

### Continuous Integration

Add these tests to your CI/CD pipeline:

```yaml
# .github/workflows/test.yml
name: Test Suite
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - name: Run tests
        run: |
          go test ./... -v -cover
          ./testing/run-tests.sh
```

---

**[⬆ Back to README](README.md)** | **[Deploy to Production →](SELF_HOST.md)**