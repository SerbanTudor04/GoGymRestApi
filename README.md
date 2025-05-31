# GoGym REST API

A comprehensive gym management REST API built with Go, PostgreSQL, and Docker. Features automatic SSL with Traefik, JWT authentication, rate limiting, and complete gym operations management.

## 🚀 Features

- **JWT Authentication** - Secure user authentication and authorization
- **Gym Management** - Create and manage multiple gym locations
- **Client Management** - Complete client lifecycle management
- **Membership System** - Flexible membership plans and assignments
- **Check-in/Check-out** - Real-time gym occupancy tracking
- **Machine Management** - Equipment tracking and assignment
- **Rate Limiting** - Built-in API protection
- **Auto SSL** - Automatic HTTPS with Let's Encrypt via Traefik
- **Health Checks** - Comprehensive monitoring endpoints
- **Docker Ready** - Complete containerization with Docker Compose

## 🏗️ Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Traefik   │────│  GoGym API  │────│ PostgreSQL  │
│ (SSL/Proxy) │    │   (Go 1.24) │    │ (15-alpine) │
└─────────────┘    └─────────────┘    └─────────────┘
```

## 📋 Prerequisites

- **Docker** and **Docker Compose**
- **Domain name** pointing to your server (for SSL)
- **Go 1.24** (for local development)

## 🚀 Quick Start

### 1. Clone and Setup

```bash
git clone https://github.com/yourusername/GoGymRestApi.git
cd GoGymRestApi

# Create deployment directory
mkdir -p /opt/gorestgym
cd /opt/gorestgym
```

### 2. Create Environment File

```bash
# Create .env file
cat > .env << EOF
# Database Configuration
DB_USER=gogymrest
DB_PASSWORD=your-secure-password-here
DB_NAME=gogym

# Let's Encrypt Email
LETSENCRYPT_EMAIL=your-email@domain.com
EOF
```

### 3. Setup Networks and Deploy

```bash
# Create Traefik network
docker network create traefik

# Start services
docker-compose up -d

# Check logs
docker-compose logs -f
```

### 4. Verify Deployment

```bash
# Test API health
curl https://demogogymapi.fintechpro.ro/api/health

# Expected response:
{
  "message": "Service is healthy",
  "data": {
    "status": "healthy",
    "timestamp": "2025-01-01T12:00:00Z",
    "database": "connected",
    "version": "1.0.0"
  }
}
```

## 📱 API Endpoints

### Authentication
```
POST /api/users/register    # Register new user
POST /api/users/login       # User login
GET  /api/users/me          # Get current user info
```

### Gym Management
```
GET  /api/gyms              # List user's gyms
POST /api/gyms/create       # Create new gym
POST /api/gyms/add-user     # Add user to gym
GET  /api/gyms/{id}/stats   # Get gym statistics
```

### Client Management
```
GET  /api/clients           # List clients
POST /api/clients/create    # Create new client
POST /api/clients/add-user  # Add user to client
POST /api/clients/checkin   # Client check-in
POST /api/clients/checkout  # Client check-out
```

### Memberships
```
GET  /api/memberships       # List available memberships
POST /api/clients/membership/add  # Add membership to client
```

### Nomenclators
```
GET  /api/nomenclators/countries     # List countries
GET  /api/nomenclators/states        # List states by country
```

## 🔐 Authentication

All endpoints (except registration, login, and health) require JWT authentication:

```bash
# Login to get token
curl -X POST https://demogogymapi.fintechpro.ro/api/users/login \
  -H "Content-Type: application/json" \
  -d '{"username": "your-username", "password": "your-password"}'

# Use token in subsequent requests
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  https://demogogymapi.fintechpro.ro/api/users/me
```

## 🏃‍♂️ Development

### Local Development Setup

```bash
# Install dependencies
go mod download

# Setup environment
cp .env.example .env
# Edit .env with your local database settings

# Run locally
go run main.go
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./server
```

### Database Setup

```bash
# Start PostgreSQL only
docker-compose up -d postgres

# Connect to database
docker exec -it postgres psql -U gogymrest -d gogym
```

## 🐳 Docker Hub

The API is automatically built and published to Docker Hub:

```bash
# Pull latest image
docker pull serbantudor/gorestgym:latest

# Run standalone
docker run -d \
  --name gorestgym-api \
  -p 8080:8080 \
  -e DB_HOST=your-db-host \
  -e DB_USER=gogymrest \
  -e DB_PASSWORD=your-password \
  -e DB_NAME=gogym \
  serbantudor/gorestgym:latest
```

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | Database hostname | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_USER` | Database username | `gogymrest` |
| `DB_PASSWORD` | Database password | `gogymrest` |
| `DB_NAME` | Database name | `gogym` |
| `DB_SSL_MODE` | SSL mode for database | `disable` |
| `SERVER_PORT` | API server port | `8080` |
| `DB_MAX_OPEN_CONNS` | Max open DB connections | `25` |
| `DB_MAX_IDLE_CONNS` | Max idle DB connections | `10` |
| `DB_MAX_LIFETIME` | Connection max lifetime | `300s` |

### Traefik Configuration

The API includes production-ready Traefik configuration with:
- **Automatic SSL** certificates via Let's Encrypt
- **Rate limiting** (10 req/sec, 20 burst)
- **Security headers** (HSTS, XSS protection, etc.)
- **CORS support** for frontend integration

## 📊 Monitoring

### Health Checks

```bash
# API health
curl https://demogogymapi.fintechpro.ro/api/health

# Container health
docker-compose ps

# Service logs
docker-compose logs gorestgym-api -f
```

### Traefik Dashboard

Access the Traefik dashboard at:
- **URL**: `https://traefik.demogogymapi.fintechpro.ro/`
- **Username**: `admin`
- **Password**: `password`

## 🔄 Updates

### Automatic Updates (CI/CD)

Updates are automatic via GitHub Actions:
1. Push to `main` branch
2. GitHub Actions builds and pushes to Docker Hub
3. Pull latest image on your server:

```bash
cd /opt/gorestgym
docker-compose pull gorestgym-api
docker-compose up -d gorestgym-api
```

### Manual Updates

```bash
# Update API only
docker pull serbantudor/gorestgym:latest
docker-compose up -d gorestgym-api

# Update all services
docker-compose pull
docker-compose up -d
```

## 🗄️ Database Schema

### Core Tables
- **users** - User accounts and authentication
- **gyms** - Gym locations and settings
- **clients** - Client information and profiles
- **memberships** - Membership plans and pricing
- **countries/states** - Geographic nomenclators

### Relationship Tables
- **user_gyms** - User-gym associations
- **user_clients** - User-client relationships
- **client_memberships** - Client membership assignments
- **client_passes** - Check-in/check-out logs

### Key Features
- **Automatic timestamps** with triggers
- **Foreign key constraints** for data integrity
- **Enum types** for status fields
- **Stored procedures** for complex operations

## 🚨 Troubleshooting

### Common Issues

#### SSL Certificate Issues
```bash
# Check Traefik logs
docker logs traefik | grep -i acme

# Verify DNS
nslookup demogogymapi.fintechpro.ro

# Test HTTP (should redirect to HTTPS)
curl -I http://demogogymapi.fintechpro.ro
```

#### Database Connection Issues
```bash
# Check database logs
docker-compose logs postgres

# Test database connection
docker exec gorestgym-api wget -qO- http://localhost:8080/api/health

# Connect to database directly
docker exec -it postgres psql -U gogymrest -d gogym
```

#### API Not Responding
```bash
# Check container status
docker-compose ps

# Check API logs
docker-compose logs gorestgym-api --tail 50

# Restart API
docker-compose restart gorestgym-api
```

### Performance Tuning

```bash
# Monitor resource usage
docker stats

# Check database performance
docker exec -it postgres psql -U gogymrest -d gogym -c "
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
ORDER BY mean_exec_time DESC 
LIMIT 10;"
```

## 🤝 Contributing

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing-feature`)
5. **Open** a Pull Request

### Development Guidelines

- Follow Go best practices and conventions
- Add tests for new functionality
- Update documentation for API changes
- Ensure Docker builds pass
- Test with PostgreSQL integration

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Gorilla Mux** - HTTP routing
- **PostgreSQL** - Database engine
- **Traefik** - Reverse proxy and SSL
- **Docker** - Containerization
- **GitHub Actions** - CI/CD pipeline

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/GoGymRestApi/issues)
- **Documentation**: [API Documentation](https://demogogymapi.fintechpro.ro/docs)
- **Email**: support@fintechpro.ro

---

**Built with ❤️ for modern gym management**