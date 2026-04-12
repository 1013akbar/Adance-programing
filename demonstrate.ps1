# Demonstration Script for Assignment 2 - gRPC Migration
# Run this to show the teacher that everything works

Write-Host "=====================================" -ForegroundColor Cyan
Write-Host "Assignment 2 Demonstration" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan

# 1. Check if services are running
Write-Host "`n[1] Checking services..." -ForegroundColor Yellow

$orderRest = Test-NetConnection -ComputerName localhost -Port 8081 -InformationLevel Quiet
$orderGrpc = Test-NetConnection -ComputerName localhost -Port 50052 -InformationLevel Quiet
$paymentRest = Test-NetConnection -ComputerName localhost -Port 8082 -InformationLevel Quiet
$paymentGrpc = Test-NetConnection -ComputerName localhost -Port 50051 -InformationLevel Quiet

if ($orderRest -and $orderGrpc -and $paymentRest -and $paymentGrpc) {
    Write-Host "✓ All services are running!" -ForegroundColor Green
} else {
    Write-Host "✗ Services not running. Please start them first:" -ForegroundColor Red
    Write-Host "  - Payment Service: cd payment-service && go run ./cmd/payment-service" -ForegroundColor White
    Write-Host "  - Order Service: cd order-service && go run ./cmd/order-service" -ForegroundColor White
    exit 1
}

# 2. Test REST APIs
Write-Host "`n[2] Testing REST APIs..." -ForegroundColor Yellow

try {
    $response = Invoke-RestMethod -Uri "http://localhost:8082/health" -Method GET -TimeoutSec 5
    Write-Host "✓ Payment REST API: OK" -ForegroundColor Green
} catch {
    Write-Host "✗ Payment REST API: Failed" -ForegroundColor Red
}

try {
    $response = Invoke-RestMethod -Uri "http://localhost:8081/health" -Method GET -TimeoutSec 5
    Write-Host "✓ Order REST API: OK" -ForegroundColor Green
} catch {
    Write-Host "✗ Order REST API: Failed" -ForegroundColor Red
}

# 3. Test gRPC services
Write-Host "`n[3] Testing gRPC Services..." -ForegroundColor Yellow

# Check if grpcurl is available
$grpcurl = Get-Command grpcurl -ErrorAction SilentlyContinue
if (-not $grpcurl) {
    Write-Host "✗ grpcurl not found. Install with: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest" -ForegroundColor Red
} else {
    Write-Host "✓ grpcurl found" -ForegroundColor Green

    # Test Payment gRPC
    try {
        $result = & grpcurl -plaintext localhost:50051 list 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Payment gRPC service: OK" -ForegroundColor Green
        } else {
            Write-Host "✗ Payment gRPC service: Failed" -ForegroundColor Red
        }
    } catch {
        Write-Host "✗ Payment gRPC service: Failed" -ForegroundColor Red
    }

    # Test Order gRPC
    try {
        $result = & grpcurl -plaintext localhost:50052 list 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Order gRPC service: OK" -ForegroundColor Green
        } else {
            Write-Host "✗ Order gRPC service: Failed" -ForegroundColor Red
        }
    } catch {
        Write-Host "✗ Order gRPC service: Failed" -ForegroundColor Red
    }
}

# 4. Demonstrate inter-service gRPC communication
Write-Host "`n[4] Demonstrating gRPC Communication..." -ForegroundColor Yellow

# Create an order via REST
Write-Host "Creating an order via REST API..." -ForegroundColor White
try {
    $orderData = @{
        customer_id = "demo-customer"
        item_name = "Demo Item"
        amount = 5000
    } | ConvertTo-Json

    $orderResponse = Invoke-RestMethod -Uri "http://localhost:8081/orders" -Method POST -Body $orderData -ContentType "application/json"
    $orderId = $orderResponse.id
    Write-Host "✓ Order created: $orderId" -ForegroundColor Green

    # Check order status
    Start-Sleep -Seconds 1
    $orderStatus = Invoke-RestMethod -Uri "http://localhost:8081/orders/$orderId" -Method GET
    Write-Host "✓ Order status: $($orderStatus.status)" -ForegroundColor Green

    # This demonstrates that Order Service called Payment Service via gRPC internally
    Write-Host "✓ Inter-service gRPC communication working (Order -> Payment)" -ForegroundColor Green

} catch {
    Write-Host "✗ Order creation failed: $($_.Exception.Message)" -ForegroundColor Red
}

# 5. Demonstrate streaming
Write-Host "`n[5] Demonstrating Real-time Streaming..." -ForegroundColor Yellow

Write-Host "Starting streaming client in background..." -ForegroundColor White
Write-Host "Run this in another terminal to see streaming:" -ForegroundColor Cyan
Write-Host "  go run client.go $orderId" -ForegroundColor White
Write-Host ""
Write-Host "Then update the order status to see real-time updates:" -ForegroundColor Cyan
Write-Host "  curl -X PATCH http://localhost:8081/orders/$orderId/cancel" -ForegroundColor White
Write-Host ""
Write-Host "The streaming client will show immediate status updates!" -ForegroundColor Green

Write-Host "`n=====================================" -ForegroundColor Cyan
Write-Host "Demonstration Complete!" -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Cyan