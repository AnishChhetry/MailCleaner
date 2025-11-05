# MailCleaner

MailCleaner is a full-stack Gmail housekeeping assistant that combines an automation-ready Go backend with a modern React frontend. It helps users triage their inbox, apply rule-based cleanups, and gain insight into sender activity through analytics dashboards.

## Features

- Google OAuth login, token storage, and session management
- Gmail synchronisation with pagination, bulk actions, and fine-grained email controls
- Rule-based cleaning with scheduled automation driven by cron-style jobs
- Trash, archive, and read/unread workflows for rapid inbox curation
- Sender analytics and subscription visibility to inform cleanup strategies

## Architecture

| Layer      | Technology Highlights |
| ---------- | --------------------- |
| Backend    | Go (Gin), PostgreSQL, Redis, Gmail API integration, background scheduler |
| Frontend   | React (Create React App), Material UI, custom API client |
| Deployment | Environment-driven configuration (`.env` for backend, optional `REACT_APP_*` for frontend) |

```
MailCleaner/
├── backend/   # Go services, API, database layer, schedulers
└── frontend/  # React SPA that consumes the backend REST API
```

## Prerequisites

- Go 1.22+ (module targets Go 1.24)
- Node.js 18+ with npm or yarn
- PostgreSQL 13+
- Redis 6+
- Google Cloud project with OAuth credentials (Web client)

## Getting Started

1. **Clone the repository**
   ```bash
   git clone https://github.com/AnishChhetry/MailCleaner.git
   cd MailCleaner
   ```
2. **Provision infrastructure**
   - Create a PostgreSQL database (e.g. `mailcleaner`).
   - Ensure Redis is running and reachable.
3. **Create Google OAuth credentials**
   - Configure an OAuth consent screen.
   - Create a Web application credential with redirect URI `http://localhost:8080/auth/google/callback`.

## Backend Setup (`backend/`)

1. Copy the sample environment file and populate the required values:
   ```bash
   cp backend/.env.example backend/.env
   ```
2. Update `backend/.env` with:
   - `POSTGRES_DSN` – PostgreSQL connection string.
   - `REDIS_URL` – Redis connection URL.
   - `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` from the Google Cloud console.
3. Install dependencies and run migrations (automatically executed on start):
   ```bash
   cd backend
   go mod download
   go run ./cmd
   ```
   The API listens on `HTTP_ADDR` (defaults to `:8080`).

### Backend Testing & Tooling

- Run unit tests:
  ```bash
  go test ./...
  ```
- Build a binary:
  ```bash
  go build -o bin/mailcleaner ./cmd
  ```

## Frontend Setup (`frontend/`)

1. Install dependencies:
   ```bash
   cd frontend
   npm install
   ```
2. (Optional) Create `frontend/.env` to point at a non-default API host:
   ```bash
   echo "REACT_APP_API_BASE=http://localhost:8080" > .env
   ```
3. Start the development server:
   ```bash
   npm start
   ```
   The app runs at `http://localhost:3000` and expects the backend at `http://localhost:8080` by default.

### Frontend Testing & Production Build

- Run tests:
  ```bash
  npm test
  ```
- Create an optimized production build:
  ```bash
  npm run build
  ```
  Deploy the generated `build/` directory behind your preferred static host.

## Environment Reference

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `HTTP_ADDR` | Backend HTTP bind address | `:8080` |
| `POSTGRES_DSN` | PostgreSQL DSN | _required_ |
| `REDIS_URL` | Redis connection URL | _required_ |
| `GOOGLE_CLIENT_ID` | Google OAuth client ID | _required_ |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret | _required_ |
| `REACT_APP_API_BASE` | Frontend API base URL override | `http://localhost:8080` |

## Operational Notes

- **Database schema**: The backend runs migrations at startup, so ensure the configured database user has schema privileges.
- **Scheduling**: Automated cleanups rely on Redis for token caching and run according to user preferences stored in the database. Keep the backend process alive to maintain the scheduler.
- **Session cookies**: The backend issues cookies scoped to `localhost`; configure HTTPS and secure cookies before production deployment.

## Troubleshooting

| Symptom | Suggested Fix |
| ------- | ------------- |
| 401 / unauthorized responses | Confirm OAuth credentials, browser cookies, and that the frontend uses the same domain/port as the backend. |
| Gmail actions fail | Ensure tokens are stored (Redis) and valid; reauthenticate through Google OAuth if necessary. |
| Scheduler not running | Keep the backend process active, verify Redis availability, and check user automation settings. |
| CORS errors | Confirm the frontend origin matches the backend CORS configuration (`http://localhost:3000` during development). |

## Next Steps

- Containerize the services for deployment.
- Configure CI to run `go test ./...` and `npm test` on every push.
- Harden production settings (HTTPS, secure cookies, logging, monitoring).
