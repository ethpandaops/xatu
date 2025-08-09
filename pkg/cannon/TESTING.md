# Cannon Package Testing Guide

This document provides comprehensive guidelines for testing the `pkg/cannon` package and serves as a reference for maintaining and extending the test suite.

## Table of Contents

1. [Testing Architecture](#testing-architecture)
2. [Mock Infrastructure](#mock-infrastructure)
3. [Testing Patterns](#testing-patterns)
4. [Test Categories](#test-categories)
5. [Performance Testing](#performance-testing)
6. [Coverage Guidelines](#coverage-guidelines)
7. [Best Practices](#best-practices)
8. [Troubleshooting](#troubleshooting)

## Testing Architecture

### Overview

The cannon package testing architecture follows a layered approach:

```
┌─────────────────────────────────────┐
│           Integration Tests         │  ← End-to-end workflows
├─────────────────────────────────────┤
│          Component Tests            │  ← Individual component testing
├─────────────────────────────────────┤
│           Mock Layer               │  ← Test doubles and utilities
├─────────────────────────────────────┤
│          Unit Tests                │  ← Pure function testing
└─────────────────────────────────────┘
```

### Key Principles

1. **Dependency Injection**: All external dependencies are abstracted through interfaces
2. **Mock-First Testing**: Comprehensive mock infrastructure enables isolated testing
3. **Test Pyramid**: Unit tests form the base, with fewer integration tests at the top
4. **Performance Awareness**: Regular performance testing prevents regressions

## Mock Infrastructure

### Available Mocks

The `pkg/cannon/mocks` package provides comprehensive mock implementations:

#### Core Mocks
- **MockTimeProvider**: Time manipulation and controlled time progression
- **MockNTPClient**: Network time synchronization simulation  
- **MockScheduler**: Job scheduling and lifecycle management
- **MockSink**: Output sink behavior simulation
- **MockBeaconNode**: Ethereum beacon node API simulation
- **MockCoordinator**: Coordination service simulation
- **MockBlockprint**: Blockprint service simulation

#### Mock Usage Example

```go
func TestWithMocks(t *testing.T) {
    // Create mocks
    timeProvider := &mocks.MockTimeProvider{}
    ntpClient := &mocks.MockNTPClient{}
    
    // Set up expectations
    fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
    timeProvider.SetCurrentTime(fixedTime)
    
    ntpResponse := &mocks.MockNTPResponse{}
    ntpResponse.SetClockOffset(10 * time.Millisecond)
    ntpClient.On("Query", "time.google.com").Return(ntpResponse, nil)
    
    // Execute test logic
    now := timeProvider.Now()
    assert.Equal(t, fixedTime, now)
    
    response, err := ntpClient.Query("time.google.com")
    require.NoError(t, err)
    assert.Equal(t, 10*time.Millisecond, response.ClockOffset())
    
    // Verify all expectations
    timeProvider.AssertExpectations(t)
    ntpClient.AssertExpectations(t)
}
```

### Mock Factory Pattern

Use the provided factory functions for consistent mock creation:

```go
// Create test configuration
config := mocks.TestConfig()

// Create test logger (suppresses output during tests)
logger := mocks.TestLogger()

// Create mock assertions helper
assertions := mocks.NewTestAssertions(t)
```

## Testing Patterns

### 1. Table-Driven Tests

Use table-driven tests for comprehensive scenario coverage:

```go
func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name        string
        config      *Config
        expectedErr string
    }{
        {
            name: "valid_config",
            config: &Config{
                Name: "test-cannon",
                Ethereum: ethereum.Config{
                    BeaconNodeAddress: "http://localhost:5052",
                },
            },
            expectedErr: "",
        },
        {
            name: "missing_name",
            config: &Config{
                Ethereum: ethereum.Config{
                    BeaconNodeAddress: "http://localhost:5052",
                },
            },
            expectedErr: "name is required",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            
            if tt.expectedErr != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedErr)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 2. Mock Expectation Pattern

Always verify mock expectations in tests:

```go
func TestWithExpectations(t *testing.T) {
    mock := &mocks.MockScheduler{}
    
    // Set up expectations
    mock.On("Start").Return()
    mock.On("IsStarted").Return(true)
    
    // Execute functionality
    mock.Start()
    started := mock.IsStarted()
    assert.True(t, started)
    
    // IMPORTANT: Always verify expectations
    mock.AssertExpectations(t)
}
```

### 3. Prometheus Metrics Testing

For components with Prometheus metrics, use registry isolation:

```go
func TestMetrics(t *testing.T) {
    // Create isolated registry to prevent conflicts
    reg := prometheus.NewRegistry()
    origRegisterer := prometheus.DefaultRegisterer
    prometheus.DefaultRegisterer = reg
    defer func() {
        prometheus.DefaultRegisterer = origRegisterer
    }()
    
    // Create metrics
    metrics := NewMetrics("test")
    
    // Test metric operations
    metrics.IncBlocksFetched("mainnet")
    
    // Verify metrics were recorded
    metricFamilies, err := reg.Gather()
    require.NoError(t, err)
    assert.NotEmpty(t, metricFamilies)
}
```

### 4. Error Path Testing

Always test error scenarios:

```go
func TestErrorHandling(t *testing.T) {
    mock := &mocks.MockNTPClient{}
    
    // Test successful case
    mock.On("Query", "valid.host").Return(&mocks.MockNTPResponse{}, nil)
    
    // Test error case
    mock.On("Query", "invalid.host").Return(nil, errors.New("connection failed"))
    
    // Test both paths
    _, err := mock.Query("valid.host")
    assert.NoError(t, err)
    
    _, err = mock.Query("invalid.host")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "connection failed")
    
    mock.AssertExpectations(t)
}
```

## Test Categories

### Unit Tests

**Purpose**: Test individual functions and methods in isolation  
**Location**: `*_test.go` files alongside source code  
**Scope**: Single function/method  
**Dependencies**: All external dependencies mocked  

**Example**:
```go
func TestConfig_Validate(t *testing.T) {
    config := &Config{Name: "test"}
    err := config.Validate()
    assert.NoError(t, err)
}
```

### Component Tests

**Purpose**: Test component interactions and behavior  
**Location**: `*_test.go` files in component packages  
**Scope**: Single component with its immediate dependencies  
**Dependencies**: External services mocked, internal dependencies real  

**Example**:
```go
func TestBeaconNode_Structure(t *testing.T) {
    config := &ethereum.Config{BeaconNodeAddress: "http://localhost:5052"}
    beaconNode := &ethereum.BeaconNode{Config: config}
    assert.Equal(t, config, beaconNode.Config)
}
```

### Integration Tests

**Purpose**: Test multiple components working together  
**Location**: `integration_test.go` files  
**Scope**: Multiple components and their interactions  
**Dependencies**: External services mocked, internal components real  

### Performance Tests

**Purpose**: Validate performance characteristics and resource usage  
**Location**: `performance_test.go` files  
**Scope**: Performance-critical paths and resource usage  
**Dependencies**: Minimal mocking to maintain realistic conditions  

**Example**:
```go
func BenchmarkMockCreation(b *testing.B) {
    for i := 0; i < b.N; i++ {
        mock := &mocks.MockTimeProvider{}
        _ = mock
    }
}
```

## Performance Testing

### Benchmarks

Write benchmarks for performance-critical code:

```go
func BenchmarkTimeProviderOperations(b *testing.B) {
    provider := &mocks.MockTimeProvider{}
    fixedTime := time.Now()
    provider.SetCurrentTime(fixedTime)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        now := provider.Now()
        _ = now
    }
}
```

### Memory Testing

Test for memory leaks and excessive allocations:

```go
func TestMemoryUsage(t *testing.T) {
    runtime.GC()
    var baseline runtime.MemStats
    runtime.ReadMemStats(&baseline)
    
    // Perform operations
    for i := 0; i < 1000; i++ {
        mock := &mocks.MockTimeProvider{}
        _ = mock
    }
    
    runtime.GC()
    var after runtime.MemStats
    runtime.ReadMemStats(&after)
    
    memoryUsed := after.Alloc - baseline.Alloc
    memoryPerOperation := memoryUsed / 1000
    
    assert.Less(t, memoryPerOperation, uint64(1024), "Memory per operation should be reasonable")
}
```

### Goroutine Leak Testing

Verify no goroutines are leaked:

```go
func TestGoroutineLeaks(t *testing.T) {
    baseline := runtime.NumGoroutine()
    
    // Perform operations that might leak goroutines
    for i := 0; i < 100; i++ {
        scheduler := &mocks.MockScheduler{}
        scheduler.On("Start").Return()
        scheduler.Start()
    }
    
    time.Sleep(100 * time.Millisecond) // Allow cleanup
    runtime.GC()
    
    final := runtime.NumGoroutine()
    assert.Less(t, final-baseline, 10, "Should not leak significant goroutines")
}
```

## Coverage Guidelines

### Coverage Targets

- **Minimum Line Coverage**: 80%
- **Target Line Coverage**: 90%+
- **Critical Path Coverage**: 100%

### Running Coverage

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./pkg/cannon/...

# View coverage report
go tool cover -html=coverage.out

# Get coverage summary
go tool cover -func=coverage.out
```

### Coverage Best Practices

1. **Focus on Critical Paths**: Ensure 100% coverage of error handling and critical business logic
2. **Ignore Generated Code**: Use build tags to exclude generated code from coverage
3. **Test Edge Cases**: Include boundary conditions and error scenarios
4. **Avoid Coverage Gaming**: Write meaningful tests, not just coverage-boosting tests

## Best Practices

### 1. Test Organization

```go
// Group related tests in the same file
// Use clear, descriptive test names
// Follow the pattern: TestComponentName_MethodName_Scenario

func TestConfig_Validate_ValidInput(t *testing.T) { /* ... */ }
func TestConfig_Validate_MissingName(t *testing.T) { /* ... */ }
func TestConfig_Validate_InvalidAddress(t *testing.T) { /* ... */ }
```

### 2. Test Data Management

```go
// Use helper functions for test data creation
func createValidConfig() *Config {
    return &Config{
        Name: "test-cannon",
        Ethereum: ethereum.Config{
            BeaconNodeAddress: "http://localhost:5052",
        },
        Coordinator: coordinator.Config{
            Address: "localhost:8081",
        },
    }
}
```

### 3. Assertion Practices

```go
// Use specific assertions
assert.Equal(t, expected, actual)           // Good
assert.True(t, expected == actual)          // Avoid

// Test error messages, not just error presence
assert.Error(t, err)                        // Good
assert.Contains(t, err.Error(), "expected") // Better
```

### 4. Mock Management

```go
// Always clean up mocks
func TestWithMocks(t *testing.T) {
    mock := &mocks.MockScheduler{}
    defer mock.AssertExpectations(t) // Ensure cleanup
    
    // Test logic...
}
```

### 5. Test Isolation

```go
// Each test should be independent
func TestIndependentTest(t *testing.T) {
    // Create fresh instances for each test
    config := createValidConfig()
    
    // Modify as needed for this specific test
    config.Name = "specific-test-name"
    
    // Test logic...
}
```

## Troubleshooting

### Common Issues

#### 1. Mock Expectation Failures

**Problem**: `FAIL: 0 out of 1 expectation(s) were met`

**Solution**: Ensure all mocked methods are called with expected parameters:

```go
// Wrong: expectation not matched
mock.On("Query", "host1").Return(response, nil)
mock.Query("host2") // Different parameter!

// Correct: parameters match
mock.On("Query", "host1").Return(response, nil)
mock.Query("host1") // Matches expectation
```

#### 2. Prometheus Registration Conflicts

**Problem**: `panic: duplicate metrics collector registration attempted`

**Solution**: Use registry isolation in tests:

```go
func TestMetrics(t *testing.T) {
    reg := prometheus.NewRegistry()
    oldReg := prometheus.DefaultRegisterer
    prometheus.DefaultRegisterer = reg
    defer func() { prometheus.DefaultRegisterer = oldReg }()
    
    // Test logic...
}
```

#### 3. Race Conditions in Tests

**Problem**: Tests fail intermittently

**Solution**: Use proper synchronization:

```go
func TestConcurrent(t *testing.T) {
    var wg sync.WaitGroup
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Test logic...
        }()
    }
    
    wg.Wait()
}
```

#### 4. Time-Dependent Test Failures

**Problem**: Tests fail due to timing issues

**Solution**: Use mock time providers:

```go
func TestTimeDependent(t *testing.T) {
    timeProvider := &mocks.MockTimeProvider{}
    fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
    timeProvider.SetCurrentTime(fixedTime)
    
    // Use timeProvider instead of time.Now()
}
```

### Debugging Tips

1. **Use t.Logf()** for debugging information:
   ```go
   t.Logf("Debug info: %v", variable)
   ```

2. **Run single tests** for focused debugging:
   ```bash
   go test -run TestSpecificTest ./pkg/cannon
   ```

3. **Enable verbose output**:
   ```bash
   go test -v ./pkg/cannon/...
   ```

4. **Use race detection**:
   ```bash
   go test -race ./pkg/cannon/...
   ```

## Maintenance

### Regular Tasks

1. **Review Coverage**: Monthly coverage reports and gap analysis
2. **Performance Monitoring**: Benchmark trending and regression detection
3. **Mock Updates**: Keep mocks synchronized with interface changes
4. **Test Cleanup**: Remove obsolete tests and update patterns

### Adding New Tests

When adding new functionality:

1. **Write Tests First**: Follow TDD when possible
2. **Update Mocks**: Extend mock interfaces as needed  
3. **Add Documentation**: Update this guide with new patterns
4. **Run Full Suite**: Ensure no regressions in existing tests

### Refactoring Tests

When refactoring:

1. **Maintain Coverage**: Ensure coverage doesn't decrease
2. **Update Patterns**: Modernize test patterns during refactoring
3. **Simplify Where Possible**: Remove unnecessary complexity
4. **Preserve Intent**: Maintain the original test purpose

---

This testing guide is a living document. Please update it as testing patterns evolve and new best practices emerge.