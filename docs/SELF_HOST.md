# Self-Hosting Guide for SUDO Kanban

Complete guide to self-hosting SUDO Kanban on your own infrastructure.

---

## 📋 Table of Contents

- [Overview](#overview)
- [Architecture Options](#architecture-options)
- [Decision Matrix](#decision-matrix)
- [Quick Start (Simple)](#quick-start-simple)
- [Production Deployment](#production-deployment)
- [Advanced Setup with nginx](#advanced-setup-with-nginx)
- [Monitoring & Observability](#monitoring--observability)
- [Security Hardening](#security-hardening)
- [Backup & Recovery](#backup--recovery)
- [Troubleshooting](#troubleshooting)

---

## Overview

SUDO Kanban is designed to be self-hosted with minimal dependencies. You can deploy it in several ways depending on your needs:

1. **Simple Deployment** - Single Docker container (perfect for personal use)
2. **Production Deployment** - nginx + load balanced instances (for teams)
3. **Advanced Deployment** - Full stack with monitoring (for organizations)

---

## Architecture Options

### Simple Architecture (Personal/Small Team)

```
┌─────────────────────────────────────────────┐
│         SIMPLE DEPLOYMENT                   │
│                                             │
│  User's Browser                             │
│     ↓                                       │
│  SUDO Go App (port 8080)                    │
│     ↓                                       │
│  Supabase (PostgreSQL)                      │
│                                             │
│  ✅Single Docker container                  │
│  ✅Auto-restart on crash                    │
│  ✅Perfect for <10 users                    │
│  ✅~256MB RAM usage                         │
└─────────────────────────────────────────────┘
```

**Pros:**
- Minimal setup
- Low resource usage
- Easy to maintain
- Fast deployment

**Cons:**
- No load balancing
- No advanced monitoring
- No automatic scaling

---

### Production Architecture (Teams/Organizations)

```
┌─────────────────────────────────────────────────────────────────┐
│           PRODUCTION DEPLOYMENT                                 │
│                                                                 │
│  Internet                                                       │
│     ↓                                                           │
│  nginx Reverse Proxy (ports 80/443)                             │
│     │                                                           │
│     ├─ SSL/TLS Termination                                      │
│     ├─ Rate Limiting                                            │
│     ├─ Static File Serving (cached)                             │
│     ├─ Gzip Compression                                         │
│     └─ Load Balancing                                           │
│            ↓                                                    │
│  ┌─────────────────────────────────────┐                        │
│  │  SUDO App Instances (3 replicas)    │                        │
│  │  ┌────────┐ ┌────────┐ ┌────────┐   │                        │
│  │  │ App #1 │ │ App #2 │ │ App #3 │   │                        │
│  │  │ :8080  │ │ :8081  │ │ :8082  │   │                        │
│  │  └────────┘ └────────┘ └────────┘   │                        │
│  └─────────────────────────────────────┘                        │
│            ↓                                                    │
│  Redis Cache (optional)                                         │
│            ↓                                                    │
│  Supabase (PostgreSQL)                                          │
│            ↓                                                    │
│  Prometheus + Grafana (monitoring)                              │
│                                                                 │
│  ✅ Load balanced                                               │
│  ✅ High availability                                           │
│  ✅ Real-time monitoring                                        │
│  ✅ Supports 100+ concurrent users                              │
└─────────────────────────────────────────────────────────────────┘
```

**Pros:**
- High availability
- Horizontal scaling
- Advanced monitoring
- Better performance

**Cons:**
- More complex setup
- Higher resource usage
- Requires more maintenance

---

## Decision Matrix

| Feature | Simple Deployment | Production Deployment |
|---------|------------------|----------------------|
| **Setup Time** | 5-10 minutes | 30-60 minutes |
| **Users Supported** | 1-10 | 10-1000+ |
| **Monthly Cost** | $5-10 | $20-100 |
| **RAM Required** | 256MB | 1-4GB |
| **CPU Required** | 0.5 core | 2-4 cores |
| **Uptime** | 99% | 99.9%+ |
| **SSL/HTTPS** | Manual or reverse proxy | Built-in (nginx) |
| **Load Balancing** | No | Yes (3 instances) |
| **Auto-scaling** | No | Manual/Docker Swarm |
| **Monitoring** | Basic logs | Prometheus + Grafana |
| **Backup Strategy** | Manual | Automated |
| **Rate Limiting** | Application level | nginx + application |
| **Static File Caching** | Go serves files | nginx (optimized) |
| **WebSocket Support** | Yes | Yes (optimized) |
| **Complexity** | 🟢 Simple | 🟡 Moderate |
| **Best For** | Personal/Small team | Teams/Organizations |

---

## Quick Start (Simple)

Perfect for personal use or small teams (<10 users).

### Prerequisites

```bash
# Required
- Docker and Docker Compose
- Supabase account (free tier works)
- Domain name (optional, can use IP)
```

### Step 1: Clone and Configure

```bash
# Clone the repository
git clone https://github.com/yourusername/sudo.git
cd sudo

# Create environment file
cp .env.example .env

# Edit .env with your details
nano .env
```

**Required environment variables:**

```bash
# Supabase Configuration
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_KEY=your-service-key-here

# Security
JWT_SECRET=your-random-32-char-secret-here
ENCRYPTION_MASTER_KEY=generate-using-command-below

# Email (Resend)
RESEND_API_KEY=your-resend-api-key
FROM_EMAIL=noreply@yourdomain.com

# Application
APP_ENV=production
PORT=8080
```

### Step 2: Generate Encryption Key

```bash
# Generate a secure encryption key
openssl rand -base64 32

# Add it to .env as ENCRYPTION_MASTER_KEY
```

### Step 3: Set Up Database

```bash
# Go to your Supabase dashboard
# SQL Editor → New Query
# Copy and paste the contents of database.sql
# Run the query
```

### Step 4: Deploy

```bash
# Build and start the container
docker build -t sudo-kanban .
docker run -d \
  --name sudo \
  -p 8080:8080 \
  --env-file .env \
  --restart unless-stopped \
  sudo-kanban

# Check if it's running
docker ps
docker logs sudo

# Access your app
# http://your-server-ip:8080
```

### Step 5: Set Up SSL (Optional but Recommended)

**Option A: Using Caddy (Easiest)**

```bash
# Install Caddy
sudo apt install caddy

# Create Caddyfile
sudo nano /etc/caddy/Caddyfile
```

```caddy
yourdomain.com {
    reverse_proxy localhost:8080
}
```

```bash
# Reload Caddy
sudo systemctl reload caddy
```

**Option B: Using Certbot + nginx**

See [Production Deployment](#production-deployment) section below.

---

## Production Deployment

For teams and organizations requiring high availability and advanced features.

### Prerequisites

```bash
# Required
- Docker and Docker Compose
- Linux server (Ubuntu 22.04 recommended)
- 2GB+ RAM, 2+ CPU cores
- Domain name with DNS configured
- SSL certificates (or Let's Encrypt)
```

### Step 1: Prepare the Server

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# Install Docker Compose
sudo apt install docker-compose-plugin -y

# Verify installation
docker --version
docker compose version
```

### Step 2: Configure Environment

```bash
# Clone repository
git clone https://github.com/yourusername/sudo.git
cd sudo

# Create production environment
cp .env.example .env.production

# Edit configuration
nano .env.production
```

**Production environment variables:**

```bash
# Supabase
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_KEY=your-service-key

# Security
JWT_SECRET=your-random-64-char-secret
ENCRYPTION_MASTER_KEY=your-encryption-key
APP_ENV=production

# Email
RESEND_API_KEY=your-api-key
FROM_EMAIL=noreply@yourdomain.com

# Application
PORT=8080

# Redis (optional, for caching)
REDIS_URL=redis://redis:6379

# Monitoring
GRAFANA_PASSWORD=your-grafana-password
```

### Step 3: SSL Certificates

**Option A: Let's Encrypt (Recommended)**

```bash
# Install certbot
sudo apt install certbot -y

# Generate certificates
sudo certbot certonly --standalone \
  -d yourdomain.com \
  -d www.yourdomain.com

# Certificates will be in /etc/letsencrypt/live/yourdomain.com/
```

**Option B: Self-Signed (Development Only)**

```bash
# Create SSL directory
mkdir -p ssl

# Generate self-signed certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout ssl/server.key \
  -out ssl/server.crt \
  -subj "/CN=yourdomain.com"
```

### Step 4: Configure nginx

The `nginx.conf` is already configured, but you may want to customize:

```bash
# Edit nginx.conf
nano nginx.conf
```

**Key configurations to check:**

```nginx
# Update server_name
server_name yourdomain.com www.yourdomain.com;

# Update SSL paths (if using Let's Encrypt)
ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;

# Adjust rate limits if needed
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
```

### Step 5: Deploy with Docker Compose

```bash
# Review docker-compose.prod.yml
cat docker-compose.prod.yml

# Start all services
docker compose -f docker-compose.prod.yml up -d

# Check status
docker compose -f docker-compose.prod.yml ps

# View logs
docker compose -f docker-compose.prod.yml logs -f app
```

### Step 6: Verify Deployment

```bash
# Test health endpoint
curl https://yourdomain.com/health

# Test SSL
curl -I https://yourdomain.com

# Check all services are running
docker compose -f docker-compose.prod.yml ps

# Expected output:
# NAME              STATE    PORTS
# sudo-app-1       running  8080/tcp
# sudo-app-2       running  8080/tcp
# sudo-app-3       running  8080/tcp
# sudo-nginx-1     running  80/tcp, 443/tcp
# sudo-redis-1     running  6379/tcp
# sudo-prometheus  running  9090/tcp
# sudo-grafana     running  3000/tcp
```

---

## Advanced Setup with nginx

### Load Balancing Configuration

The included `nginx.conf` provides:

**1. Round-Robin Load Balancing**
```nginx
upstream app_servers {
    least_conn;  # Route to server with fewest connections
    server app:8080 max_fails=3 fail_timeout=30s;
    keepalive 32;
}
```

**2. Rate Limiting**
```nginx
# API endpoints: 10 requests/second
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;

# WebSocket: 5 connections/second
limit_req_zone $binary_remote_addr zone=websocket:10m rate=5r/s;
```

**3. Security Headers**
```nginx
add_header X-Frame-Options "SAMEORIGIN" always;
add_header X-Content-Type-Options "nosniff" always;
add_header Strict-Transport-Security "max-age=63072000" always;
```

**4. Gzip Compression**
```nginx
gzip on;
gzip_types text/plain text/css application/json application/javascript;
```

### Scaling Instances

Edit `docker-compose.prod.yml` to adjust replica count:

```yaml
services:
  app:
    deploy:
      replicas: 5  # Increase from 3 to 5
```

Reload:
```bash
docker compose -f docker-compose.prod.yml up -d --scale app=5
```

---

## Monitoring & Observability

### Prometheus Metrics

Access Prometheus at `http://your-server:9090`

**Key metrics to monitor:**
- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - Request latency
- `websocket_connections` - Active WebSocket connections
- `go_goroutines` - Number of goroutines
- `go_memstats_alloc_bytes` - Memory usage

### Grafana Dashboards

Access Grafana at `http://your-server:3000` (admin/your-password)

**Pre-configured dashboards:**
1. Application Overview
2. WebSocket Performance
3. Database Queries
4. System Resources

### Application Logs

```bash
# View real-time logs
docker compose -f docker-compose.prod.yml logs -f app

# View specific service
docker compose -f docker-compose.prod.yml logs -f nginx

# Export logs
docker compose -f docker-compose.prod.yml logs --no-color > sudo-logs.txt
```

---

## Security Hardening

### 1. Firewall Configuration

```bash
# Using UFW (Ubuntu)
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# Verify
sudo ufw status
```

### 2. Database Security

**Supabase Settings:**
- Enable Row Level Security (RLS) on all tables
- Use service key only on backend (never expose in frontend)
- Set up database backups
- Enable Point-in-Time Recovery

### 3. Encryption

**Ensure these are set:**
```bash
# Generate strong keys
ENCRYPTION_MASTER_KEY=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -base64 48)
```

**Never commit these to git!**

### 4. Regular Updates

```bash
# Update Docker images
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d

# Update system packages
sudo apt update && sudo apt upgrade -y
```

### 5. SSL/TLS Best Practices

The nginx configuration already includes:
- TLS 1.2 and 1.3 only
- Strong cipher suites
- HSTS headers
- Secure session caching

**Auto-renew Let's Encrypt:**
```bash
# Certbot auto-renewal (runs twice daily)
sudo certbot renew --dry-run

# Add to crontab if not automatic
sudo crontab -e
0 0 * * * certbot renew --post-hook "docker compose -f /path/to/sudo/docker-compose.prod.yml restart nginx"
```

---

## Backup & Recovery

### Database Backups

**Supabase automatic backups:**
- Free tier: Daily backups (7 days retention)
- Pro tier: Point-in-time recovery

**Manual backup:**
```bash
# Export database
pg_dump -h your-db.supabase.co \
  -U postgres \
  -d postgres \
  --clean --if-exists \
  > sudo_backup_$(date +%Y%m%d).sql

# Restore
psql -h your-db.supabase.co \
  -U postgres \
  -d postgres \
  < sudo_backup_20250119.sql
```

### Application Data Backup

```bash
# Backup environment files
tar -czf sudo-config-backup.tar.gz \
  .env.production \
  nginx.conf \
  docker-compose.prod.yml

# Backup logs
docker compose -f docker-compose.prod.yml logs --no-color > logs-backup.txt

# Backup volumes
docker run --rm \
  -v sudo_redis_data:/data \
  -v $(pwd):/backup \
  alpine tar -czf /backup/redis-backup.tar.gz /data
```

### Disaster Recovery Plan

1. **Regular backups** - Daily database, weekly configs
2. **Offsite storage** - Store backups in different location/cloud
3. **Test restores** - Monthly restore tests
4. **Documentation** - Keep recovery procedures updated
5. **Monitoring alerts** - Get notified of failures

---

## Troubleshooting

### Common Issues

#### 1. App Won't Start

**Check logs:**
```bash
docker compose -f docker-compose.prod.yml logs app
```

**Common causes:**
- Missing environment variables
- Database connection failure
- Port already in use

**Solution:**
```bash
# Check env file
cat .env.production

# Test database connection
docker compose -f docker-compose.prod.yml run --rm app \
  wget -O- http://your-db.supabase.co

# Check port availability
sudo lsof -i :8080
```

#### 2. SSL Certificate Errors

**Check certificate:**
```bash
openssl s_client -connect yourdomain.com:443 -servername yourdomain.com
```

**Renew Let's Encrypt:**
```bash
sudo certbot renew --force-renewal
docker compose -f docker-compose.prod.yml restart nginx
```

#### 3. High Memory Usage

**Check resource usage:**
```bash
docker stats

# Limit container memory
docker compose -f docker-compose.prod.yml up -d --scale app=2
```

**Optimize:**
- Reduce number of replicas
- Enable Redis caching
- Optimize database queries

#### 4. WebSocket Connection Failures

**Check nginx config:**
```bash
# Verify WebSocket proxy settings
grep -A 10 "location /ws/" nginx.conf
```

**Test WebSocket:**
```bash
# Using wscat
npm install -g wscat
wscat -c wss://yourdomain.com/ws/your-board-id
```

#### 5. Slow Performance

**Check Prometheus metrics:**
- High response times
- Database query performance
- Memory/CPU usage

**Optimize:**
- Enable Redis caching
- Scale up instances
- Optimize database indexes
- Enable gzip compression (already in nginx.conf)

---

## Performance Tuning

### nginx Tuning

```nginx
# Increase worker processes
worker_processes auto;

# Increase worker connections
events {
    worker_connections 2048;
}

# Adjust buffer sizes
client_body_buffer_size 128k;
client_max_body_size 10m;
```

### Go Application Tuning

```bash
# Increase max file descriptors
ulimit -n 65536

# Set GOMAXPROCS
export GOMAXPROCS=$(nproc)
```

### Database Tuning

**Supabase:**
- Add indexes on frequently queried columns
- Use connection pooling
- Enable query performance insights
- Upgrade to larger instance if needed

---

## Cost Estimates

### Simple Deployment (~$10/month)

- **Compute**: $5/month (DigitalOcean Droplet, 512MB)
- **Database**: Free (Supabase free tier)
- **Domain**: $12/year (~$1/month)
- **Total**: ~$6-10/month

### Production Deployment (~$50/month)

- **Compute**: $24/month (DigitalOcean Droplet, 2GB, 2 CPU)
- **Database**: $25/month (Supabase Pro)
- **Domain**: $12/year (~$1/month)
- **Total**: ~$50/month

### Enterprise Deployment (~$200/month)

- **Compute**: $100/month (DigitalOcean Droplet, 8GB, 4 CPU)
- **Database**: $95/month (Supabase Team)
- **Domain**: $12/year
- **Monitoring**: Free (self-hosted Prometheus/Grafana)
- **Total**: ~$200/month

---

## Migration from Railway/Cloud

If you're currently using Railway or another PaaS:

### 1. Export Data

```bash
# Backup Supabase database
pg_dump <connection-string> > railway-backup.sql
```

### 2. Deploy Self-Hosted

Follow the [Production Deployment](#production-deployment) guide above.

### 3. Import Data

```bash
# Import to your self-hosted Supabase
psql <new-connection-string> < railway-backup.sql
```

### 4. Update DNS

```bash
# Point your domain to new server
A record: yourdomain.com → your-server-ip
```

### 5. Test & Verify

```bash
# Run health checks
curl https://yourdomain.com/health

# Test functionality
# - Login
# - Create board
# - Add task
# - WebSocket updates
```

### 6. Decommission Old Server

Once verified working:
```bash
# Keep old server running for 24h as backup
# Monitor logs for any issues
# Then delete Railway/cloud deployment
```

---

## Compliance & Legal

### GDPR Compliance

SUDO Kanban's self-hosted nature helps with GDPR compliance:

- ✅ **Data Control**: You control all data
- ✅ **Right to Erasure**: Delete user data with account deletion feature
- ✅ **Data Portability**: Export data from Supabase
- ✅ **Encryption**: All sensitive data encrypted at rest

### License Compliance

Remember the **MIT License with Commons Clause**:

✅ **You CAN:**
- Self-host for personal/organization use
- Modify the code
- Run for internal teams

❌ **You CANNOT:**
- Offer as a paid service to others
- Sell hosting services
- Create competing SaaS

For commercial licensing, contact the project maintainer.

---

## Support & Resources

### Documentation

- [README.md](README.md) - Project overview and quick start
- [SECURITY.md](SECURITY.md) - Security implementation details
- [TESTING_SETUP_GUIDE.md](TESTING_SETUP_GUIDE.md) - Testing and validation
- [learning/](learning/) - Technical deep dives

### Community

- GitHub Issues - Bug reports and feature requests
- GitHub Discussions - Q&A and community support

### Professional Support

For commercial support, custom features, or consulting:
- Contact: [your-email]
- Commercial licensing available

---

## Next Steps

After deployment:

1. ✅ **Test thoroughly** - Run through all features
2. ✅ **Set up monitoring** - Configure alerts
3. ✅ **Enable backups** - Automate database backups
4. ✅ **Document setup** - Keep notes for your team
5. ✅ **Plan maintenance** - Schedule update windows
6. ✅ **Monitor costs** - Track resource usage

---

**You're now running SUDO Kanban on your own infrastructure!** 🎉

For questions or issues, check the [Troubleshooting](#troubleshooting) section or open a GitHub issue.
