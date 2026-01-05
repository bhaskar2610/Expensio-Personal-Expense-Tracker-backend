# Expensio Backend

Go backend for Expensio expense tracker.

## Setup

1. Install PostgreSQL

2. Create database:
```bash
createdb expensio
```

3. Copy `.env.example` to `.env` and update values:
```bash
cp .env.example .env
```

4. Install dependencies:
```bash
go mod download
```

5. Run the server:
```bash
go run cmd/server/main.go
```

The API will be available at `http://localhost:8080`

## API Endpoints

### Authentication
- `POST /auth/register` - Register new user
- `POST /auth/login` - Login user

### Expenses (Protected)
- `GET /expenses` - Get all expenses (supports ?month=01&year=2024)
- `POST /expenses` - Create expense
- `PUT /expenses/:id` - Update expense
- `DELETE /expenses/:id` - Delete expense

### Trips (Protected)
- `GET /trips` - Get all trips
- `POST /trips` - Create trip
- `GET /trips/:id` - Get trip with expenses
- `POST /trips/:id/expenses` - Add expense to trip

### Dashboard (Protected)
- `GET /dashboard/summary` - Get monthly summary and recent expenses
- `GET /dashboard/pie-chart` - Get category-wise breakdown (supports ?month=01&year=2024)

### Health
- `GET /health` - Health check

## Database Schema

See models in `internal/models/`

