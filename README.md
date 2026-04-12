# Advanced Programming 2 - Assignment 1

## Quick Links
- 🎯 **Getting Started**: See [Quick Start](#quick-start-windows) below
- 🧪 **Testing**: See [TESTING_GUIDE.md](TESTING_GUIDE.md) for comprehensive test scenarios
- 📋 **Submission Checklist**: See [SUBMISSION_CHECKLIST.md](SUBMISSION_CHECKLIST.md) for defense prep
- 🎨 **Architecture**: [architecture-diagram.svg](architecture-diagram.svg)

## Student
- Name: Akbar Khalili
- Course: Advanced Programming 2
- Topic: Clean Architecture based Microservices (Order & Payment)

## Implemented Scope
- Two microservices in Go:
  - `order-service`
  - `payment-service`
- REST communication only (Gin)
- Clean Architecture layering inside each service
- Separate PostgreSQL database per service
- Timeout-based synchronous HTTP integration
- Failure handling when Payment Service is unavailable
- Bonus: Idempotency support through `Idempotency-Key`

## Bounded Contexts
1. Order Context
- Owns order lifecycle state: `Pending`, `Paid`, `Failed`, `Cancelled`
- Owns customer/order attributes
- Calls Payment Service for authorization

2. Payment Context
- Owns payment authorization and transaction records
- Enforces payment limit rule
- Never reads or writes order tables

## Clean Architecture Mapping
Each service follows this structure:

- `internal/domain`: Entity and pure domain constants
- `internal/usecase`: Business logic, invariants, and ports (interfaces)
- `internal/repository`: PostgreSQL implementation of repository ports
- `internal/transport/http`: Thin Gin handlers and routing
- `internal/app`: Infrastructure adapters (outbound payment REST client in order-service)
- `cmd/<service>/main.go`: Composition root with manual dependency injection

## Business Rules Covered
1. Money is `int64` (cents) only
2. Amount must be `> 0`
3. Paid orders cannot be cancelled (only Pending can be cancelled)
4. Payment limit:
   - if `amount > 100000`, payment status is `Declined`
5. Order Service uses `http.Client{Timeout: 2 * time.Second}`

## Failure Handling Choice
If Payment Service is unavailable:
1. Order Service request does not hang (timeout max 2 seconds)
2. API returns `503 Service Unavailable`
3. Already-created order is marked as `Failed`

Reasoning:
- Keeps order state explicit and observable for operators/clients
- Avoids indefinite pending records after integration failure

## Architecture Diagram
```mermaid
flowchart LR
    Client --> OrderAPI[Order Service API]
    OrderAPI --> OrderUC[Order Use Cases]
    OrderUC --> OrderRepo[Order Repository]
    OrderRepo --> OrderDB[(Order DB)]

    OrderUC --> PaymentClient[REST Payment Client with 2s timeout]
    PaymentClient --> PaymentAPI[Payment Service API]
    PaymentAPI --> PaymentUC[Payment Use Cases]
    PaymentUC --> PaymentRepo[Payment Repository]
    PaymentRepo --> PaymentDB[(Payment DB)]
```

## Running Locally

### Prerequisites
- Go 1.22+
- PostgreSQL 14+ (running on localhost:5432)
- psql command-line tool

### Quick Start (Windows)

#### 1. Setup PostgreSQL databases and users

Run from project root:
```powershell
# Set postgres password
$env:PGPASSWORD = "postgres"

# Create users and databases
Get-Content "setup.sql" | & "C:\Program Files\PostgreSQL\16\bin\psql.exe" -U postgres -h localhost

# Apply migrations to order database
$env:PGPASSWORD = "1234"
Get-Content "apply_order_migrations.sql" | & "C:\Program Files\PostgreSQL\16\bin\psql.exe" -U order_user -h localhost -d order_db

# Apply migrations to payment database
Get-Content "apply_payment_migrations.sql" | & "C:\Program Files\PostgreSQL\16\bin\psql.exe" -U payment_user -h localhost -d payment_db
```

#### 2. Run payment service
```powershell
cd payment-service
go mod tidy
go run ./cmd/payment-service
```
Payment service runs on: `http://localhost:8082`

#### 3. Run order service (in new terminal)
```powershell
cd order-service
go mod tidy
go run ./cmd/order-service
```
Order service runs on: `http://localhost:8081`

### Docker (Alternative)

```bash
docker compose up --build
```

### Database Credentials

| Service | User | Password | Database | Port |
|---------|------|----------|----------|------|
| Order | order_user | 1234 | order_db | 5432 |
| Payment | payment_user | 1234 | payment_db | 5432 |

## API Testing

### Option 1: Web Frontend (Recommended for Demo)
Open `frontend.html` in your browser to have an interactive UI for testing all endpoints.

**Features:**
- Live API testing with visual response formatting
- Automatic Order ID capture from create responses
- Color-coded status indicators
- Test flow recommendations
- Mobile-friendly interface

```bash
# Simply open in browser:
./frontend.html
```

### Option 2: Postman Collection
Import `AP2_Assignment1.postman_collection.json` into Postman:
1. Download Postman from https://www.postman.com/downloads/
2. File → Import → Select `AP2_Assignment1.postman_collection.json`
3. Set environment variables:
   - `order_service_url`: `localhost:8081`
   - `payment_service_url`: `localhost:8082`
4. Run requests or entire collection

**Includes:**
- All CRUD operations for orders and payments
- Integration test scenarios
- Automated response validation
- Idempotency key handling

### Option 3: curl (Command Line)

#### Order Service Endpoints

**Create Order (Basic)**
```bash
curl -X POST http://localhost:8081/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust-001",
    "item_name": "Laptop",
    "amount": 15000
  }'
```

**Create Order (With Idempotency Key)**
```bash
curl -X POST http://localhost:8081/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{
    "customer_id": "cust-001",
    "item_name": "Laptop",
    "amount": 15000
  }'
```

**Response (201 Created - Order Paid)**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": "cust-001",
  "item_name": "Laptop",
  "amount": 15000,
  "status": "Paid",
  "created_at": "2026-04-05T10:30:00Z"
}
```

**Get Order**
```bash
curl http://localhost:8081/orders/{order_id}
```

**Cancel Order (Only Pending)**
```bash
curl -X PATCH http://localhost:8081/orders/{order_id}/cancel
```

#### Payment Service Endpoints

**Create Payment (Amount < 100000 - Authorized)**
```bash
curl -X POST http://localhost:8082/payments \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "ord-001",
    "amount": 50000
  }'
```

**Response (201 Created)**
```json
{
  "id": "payment-001",
  "order_id": "ord-001",
  "transaction_id": "txn-550e8400",
  "amount": 50000,
  "status": "Authorized"
}
```

**Create Payment (Amount > 100000 - Declined)**
```bash
curl -X POST http://localhost:8082/payments \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "ord-002",
    "amount": 150001
  }'
```

**Response (201 Created - but declined)**
```json
{
  "status": "Declined",
  "amount": 150001
}
```

**Get Payment by Order ID**
```bash
curl http://localhost:8082/payments/{order_id}
```

#### Test Scenarios

**Scenario 1: Successful Payment Flow**
```bash
# 1. Create order with amount < 100000
curl -X POST http://localhost:8081/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"cust-1","item_name":"Laptop","amount":15000}'
# Expected: Order with status "Paid"

# 2. Get the order
curl http://localhost:8081/orders/{order_id}
# Expected: Order status is "Paid"

# 3. Try to cancel (should fail)
curl -X PATCH http://localhost:8081/orders/{order_id}/cancel
# Expected: 409 Conflict - "cannot cancel paid order"
```

**Scenario 2: Declined Payment (Exceeds Limit)**
```bash
# Create order with amount > 100000
curl -X POST http://localhost:8081/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"cust-2","item_name":"Mercedes","amount":150001}'
# Expected: Order with status "Failed"
```

**Scenario 3: Idempotency Testing**
```bash
# Send same request twice with same Idempotency-Key
# Should return 201 first time, 200 second time (already processed)
curl -X POST http://localhost:8081/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: order-idem-123" \
  -d '{"customer_id":"cust-3","item_name":"Phone","amount":50000}'

# Same request again
curl -X POST http://localhost:8081/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: order-idem-123" \
  -d '{"customer_id":"cust-3","item_name":"Phone","amount":50000}'
# Expected: Returns same order (no duplicate)
```

**Scenario 4: Service Unavailability (Manual Test)**
1. Stop Payment Service: Ctrl+C in payment service terminal
2. Try to create order:
   ```bash
   curl -X POST http://localhost:8081/orders \
     -H "Content-Type: application/json" \
     -d '{"customer_id":"cust-4","item_name":"Tablet","amount":30000}'
   ```
3. Expected: 503 Service Unavailable (after 2-second timeout)
4. Order will be marked as "Failed"

### PowerShell Examples (Windows)

```powershell
# Create order (authorized)
$auth_body = @{customer_id="cust-win-1"; item_name="Laptop"; amount=15000} | ConvertTo-Json
Invoke-WebRequest -Uri "http://localhost:8081/orders" `
  -Method POST `
  -ContentType "application/json" `
  -Body $auth_body

# Create order (declined)
$decline_body = @{customer_id="cust-win-2"; item_name="Mercedes"; amount=150001} | ConvertTo-Json
Invoke-WebRequest -Uri "http://localhost:8081/orders" `
  -Method POST `
  -ContentType "application/json" `
  -Body $decline_body

# Get order
Invoke-WebRequest -Uri "http://localhost:8081/orders/{order_id}" -Method GET

# Cancel order
Invoke-WebRequest -Uri "http://localhost:8081/orders/{order_id}/cancel" -Method PATCH

# Create payment
$payment_body = @{order_id="ord-001"; amount=50000} | ConvertTo-Json
Invoke-WebRequest -Uri "http://localhost:8082/payments" `
  -Method POST `
  -ContentType "application/json" `
  -Body $payment_body

# Get payment
Invoke-WebRequest -Uri "http://localhost:8082/payments/{order_id}" -Method GET
```

## Submission
Zip the whole project folder as:
- `AP2_Assignment1_name_surname_group.zip`

### Included in Submission
- ✅ Both service source code (clean architecture implementation)
- ✅ SQL migration scripts for each service
- ✅ README.md (this file with comprehensive documentation)
- ✅ Architecture diagram (in README and mermaid format)
- ✅ **frontend.html** - Interactive web UI for API testing
- ✅ **AP2_Assignment1.postman_collection.json** - Postman collection for testing
- ✅ docker-compose.yml - Docker setup (optional)
- ✅ run.ps1 - PowerShell startup script for Windows
- ✅ Database migration scripts

### For Defense Presentation

**Prepare to demonstrate:**
1. Run the frontend.html and show:
   - Create successful order (Paid)
   - Create failed order (Declined - exceeds limit)
   - Cancel pending order (works)
   - Try to cancel paid order (fails with 409 Conflict)

2. Use Postman collection to show:
   - All CRUD endpoints working
   - Request/response examples
   - Idempotency in action
   - Error handling (503 for unavailable service)

3. Show code quality:
   - Clean architecture layers (domain, usecase, repository, transport)
   - No shared database between services
   - Timeout implementation (2 seconds)
   - Business rule enforcement

**Questions to prepare for:**
- Why use int64 for money? (Answer: Avoid floating-point precision errors)
- How does idempotency prevent duplicate orders? (Answer: Check Idempotency-Key header)
- What happens if Payment Service is down? (Answer: Returns 503, marks order as Failed)
- Why each service has its own database? (Answer: Microservices principle - bounded contexts)
- How is the timeout enforced? (Answer: http.Client{Timeout: 2*time.Second})
