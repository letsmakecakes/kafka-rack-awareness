@echo off
REM Windows Batch Script for Running Kafka Rack Awareness Tests

echo === Kafka Rack Awareness Test Suite ===
echo.

REM Check if Docker is running
docker info >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Docker is not running. Please start Docker and try again.
    exit /b 1
)
echo [OK] Docker is running

REM Clean up existing containers
echo [INFO] Cleaning up existing containers...
docker-compose down -v >nul 2>&1
echo [OK] Cleanup complete

REM Start Kafka cluster
echo [INFO] Starting Kafka cluster with rack awareness...
docker-compose up -d
if errorlevel 1 (
    echo [ERROR] Failed to start Kafka cluster
    exit /b 1
)

REM Wait for Kafka to be ready
echo [INFO] Waiting for Kafka cluster to be ready (30 seconds)...
timeout /t 30 /nobreak >nul

REM Verify brokers are accessible
echo [INFO] Verifying broker accessibility...
for %%p in (9092 9093 9094) do (
    netstat -an | findstr ":%%p" >nul
    if errorlevel 1 (
        echo [WARNING] Broker on port %%p may not be ready yet
    ) else (
        echo [OK] Broker on port %%p is accessible
    )
)

REM Initialize Go modules
echo [INFO] Initializing Go modules...
go mod tidy
if errorlevel 1 (
    echo [ERROR] Failed to initialize Go modules
    exit /b 1
)
echo [OK] Go modules initialized

REM Run tests
echo [INFO] Running Kafka Rack Awareness Tests...
echo.

go test -v -timeout 10m
if errorlevel 1 (
    echo.
    echo =========================================
    echo [FAILED] Some tests failed
    echo =========================================
    set TEST_RESULT=FAILED
) else (
    echo.
    echo =========================================
    echo [SUCCESS] All tests passed successfully!
    echo =========================================
    set TEST_RESULT=PASSED
)

REM Cleanup prompt
echo.
set /p CLEANUP="Do you want to stop the Kafka cluster? (Y/N): "
if /i "%CLEANUP%"=="Y" (
    echo [INFO] Stopping Kafka cluster...
    docker-compose down -v
    echo [OK] Kafka cluster stopped
) else (
    echo [INFO] Kafka cluster is still running. Use 'docker-compose down -v' to stop it.
)

echo.
echo Test run complete! Result: %TEST_RESULT%

if "%TEST_RESULT%"=="FAILED" exit /b 1
exit /b 0
