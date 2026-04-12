#!/usr/bin/env pwsh
# Quick-start script for Advanced Programming 2 - Order & Payment Microservices
# Run this from the project root to set up and start both services

Write-Host "=====================================" -ForegroundColor Cyan
Write-Host "Advanced Programming 2 - Quick Start" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan

# Setup PostgreSQL
Write-Host "`n[1/3] Setting up PostgreSQL databases and users..." -ForegroundColor Yellow

$pgPath = "C:\Program Files\PostgreSQL\16\bin\psql.exe"
if (-not (Test-Path $pgPath)) {
    Write-Host "ERROR: PostgreSQL not found at $pgPath" -ForegroundColor Red
    Write-Host "Please install PostgreSQL 16 and try again." -ForegroundColor Red
    exit 1
}

# Setup databases
$env:PGPASSWORD = "1234"
Get-Content "setup.sql" | & $pgPath -U postgres -h localhost -q 2>&1 | Select-String -NotMatch "^$"
Write-Host "✓ Databases and users created" -ForegroundColor Green

# Apply migrations
$env:PGPASSWORD = "1234"
Write-Host "  Applying order migrations..." -NoNewline
Get-Content "apply_order_migrations.sql" | & $pgPath -U order_user -h localhost -d order_db -q 2>&1 | Select-String -NotMatch "^$"
Write-Host " ✓" -ForegroundColor Green

Write-Host "  Applying payment migrations..." -NoNewline
Get-Content "apply_payment_migrations.sql" | & $pgPath -U payment_user -h localhost -d payment_db -q 2>&1 | Select-String -NotMatch "^$"
Write-Host " ✓" -ForegroundColor Green

# Start payment service
Write-Host "`n[2/3] Starting payment service..." -ForegroundColor Yellow
Set-Location "payment-service"
go mod tidy | Out-Null
Start-Process pwsh -ArgumentList "-NoExit", "-Command", "go run ./cmd/payment-service"
Write-Host "✓ Payment service started on http://localhost:8082" -ForegroundColor Green

Start-Sleep -Seconds 2

# Start order service
Write-Host "`n[3/3] Starting order service..." -ForegroundColor Yellow
Set-Location "..\order-service"
go mod tidy | Out-Null
Start-Process pwsh -ArgumentList "-NoExit", "-Command", "go run ./cmd/order-service"
Write-Host "✓ Order service started on http://localhost:8081" -ForegroundColor Green

Write-Host "`n=====================================" -ForegroundColor Cyan
Write-Host "✓ System is running!" -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Services:" -ForegroundColor Cyan
Write-Host "  Order Service:   http://localhost:8081" -ForegroundColor White
Write-Host "  Payment Service: http://localhost:8082" -ForegroundColor White
Write-Host ""
Write-Host "Test endpoints:" -ForegroundColor Cyan
Write-Host "  Create order:  POST http://localhost:8081/orders" -ForegroundColor White
Write-Host "  Get order:     GET  http://localhost:8081/orders/{id}" -ForegroundColor White
Write-Host "  Cancel order:  PATCH http://localhost:8081/orders/{id}/cancel" -ForegroundColor White
Write-Host ""
Write-Host "Both services will run in separate terminal windows." -ForegroundColor Yellow
Write-Host "Close the terminal windows to stop the services." -ForegroundColor Yellow

Set-Location ".."
