# Pkg/Cannon Unit Test Implementation Plan

## Overview
> This plan outlines the comprehensive addition of unit tests to the pkg/cannon package and all its child directories. The cannon package is a critical component responsible for backfilling and processing Ethereum beacon chain data. Currently, it has zero unit test coverage, which poses significant risks for maintenance and feature development. This implementation will establish a robust testing foundation with strong coverage and ensure proper dependency injection for testability.

## Current State Assessment

### Existing Implementation Analysis
- **Zero test coverage**: No `_test.go` files exist in the entire pkg/cannon directory tree
- **Complex dependency graph**: Heavy reliance on external services (beacon API, coordinator gRPC, blockprint HTTP API)
- **Monolithic constructors**: Large initialization functions with many hard-coded dependencies
- **Background processing**: Multiple goroutines with timing-dependent behavior
- **State management**: Complex iterator and backfilling logic with persistent state
- **Interface gaps**: Missing interface abstractions for key external dependencies

### Identified Limitations
- **Untestable constructors**: `Cannon.New()` and related constructors are tightly coupled to external services
- **Hard-to-mock dependencies**: Direct instantiation of HTTP/gRPC clients
- **Complex lifecycle management**: Start/stop sequences with interdependent components
- **Time-dependent logic**: NTP sync, backoff strategies, and scheduled processing
- **State persistence**: Iterator logic tightly coupled to coordinator service

### Technical Debt
- **Missing interfaces**: No abstractions for beacon client, coordinator, or blockprint services
- **Monolithic functions**: Large functions handling multiple responsibilities
- **Hard-coded timeouts**: Fixed durations making timing-based tests difficult
- **Global state**: Some components rely on package-level configuration

## Goals

1. **Primary goal**: Achieve comprehensive unit test coverage (>90%) for all pkg/cannon components with the ability to run `go test ./pkg/cannon/...` successfully
2. **Testability refactoring**: Restructure components to support dependency injection and interface-based testing
3. **Mock infrastructure**: Establish comprehensive mocking framework for external dependencies
4. **Test organization**: Create logical test structure that mirrors the component architecture
5. **CI/CD integration**: Ensure tests run reliably in automated environments
6. **Documentation**: Provide clear testing patterns and examples for future development

### Non-functional Requirements
- **Performance**: Tests should complete within 30 seconds for the entire suite
- **Reliability**: Tests must be deterministic and not dependent on external services
- **Maintainability**: Test code should be as maintainable as production code
- **Coverage**: Minimum 90% line coverage, 85% branch coverage

## Design Approach

### Architecture Overview
The testing strategy follows a layered approach with clear separation of concerns:
- **Interface abstraction layer**: Define interfaces for all external dependencies
- **Mock implementation layer**: Provide comprehensive mocks for all interfaces
- **Unit test layer**: Test individual components in isolation
- **Integration test helpers**: Utilities for testing component interactions

The approach prioritizes dependency injection and interface-based design to enable comprehensive unit testing without external service dependencies.

### Component Breakdown

1. **TestableCannonFactory**
   - Purpose: Factory pattern for creating testable Cannon instances
   - Responsibilities: Dependency injection, configuration setup, mock integration
   - Interfaces: Provides builder pattern for test setup

2. **MockServiceLayer** 
   - Purpose: Comprehensive mocking of external services
   - Responsibilities: Beacon API mocking, coordinator mocking, blockprint mocking
   - Interfaces: Implements all service interfaces with configurable responses

3. **TestUtilities**
   - Purpose: Common testing utilities and helpers
   - Responsibilities: Test data generation, assertion helpers, test lifecycle management
   - Interfaces: Provides reusable testing patterns

4. **ComponentTestSuites**
   - Purpose: Organized test suites for each major component
   - Responsibilities: Comprehensive testing of individual components
   - Interfaces: Follows standard Go testing patterns with testify integration

## Implementation Approach

### 1. Interface Abstraction and Dependency Injection

#### Specific Changes
- Create interface abstractions for all external dependencies
- Refactor constructors to accept interfaces instead of concrete types
- Implement builder pattern for complex component initialization
- Extract configuration validation into pure functions

#### Files Affected
- `pkg/cannon/cannon.go` - Main component refactoring
- `pkg/cannon/ethereum/beacon.go` - Beacon client interface
- `pkg/cannon/coordinator/client.go` - Coordinator interface  
- `pkg/cannon/blockprint/client.go` - Blockprint interface
- New files: `interfaces.go`, `factory.go`, `builder.go`

#### Sample Implementation
```go
// interfaces.go - Define core interfaces
type BeaconNodeInterface interface {
    GetBeaconBlock(ctx context.Context, identifier string) (*spec.VersionedSignedBeaconBlock, error)
    Synced(ctx context.Context) (bool, error)
    Start(ctx context.Context) error
    GetSpecialForkSchedule() (*ForkSchedule, error)
}

type CoordinatorInterface interface {
    GetCannonLocation(ctx context.Context, req *xatu.GetCannonLocationRequest) (*xatu.GetCannonLocationResponse, error)
    UpsertCannonLocationRequest(ctx context.Context, req *xatu.UpsertCannonLocationRequest) error
}

// factory.go - Testable factory pattern
type CannonFactory struct {
    config        *Config
    beaconNode    BeaconNodeInterface
    coordinator   CoordinatorInterface
    blockprint    BlockprintInterface
    logger        logrus.FieldLogger
}

func NewTestableCannonFactory() *CannonFactory {
    return &CannonFactory{}
}

func (f *CannonFactory) WithBeaconNode(bn BeaconNodeInterface) *CannonFactory {
    f.beaconNode = bn
    return f
}

func (f *CannonFactory) WithCoordinator(coord CoordinatorInterface) *CannonFactory {
    f.coordinator = coord
    return f
}

func (f *CannonFactory) Build() (*Cannon, error) {
    return &Cannon{
        config:      f.config,
        beaconNode:  f.beaconNode,
        coordinator: f.coordinator,
        blockprint:  f.blockprint,
        log:         f.logger,
    }, nil
}
```

### 2. Mock Infrastructure Development

#### Specific Changes
- Create comprehensive mocks for all interfaces using testify/mock
- Implement configurable mock responses for different test scenarios
- Add mock data generators for realistic test data
- Create mock lifecycle management utilities

#### Files Affected
- New directory: `pkg/cannon/mocks/`
- `mocks/beacon_node_mock.go` - Beacon API mocking
- `mocks/coordinator_mock.go` - Coordinator service mocking
- `mocks/blockprint_mock.go` - Blockprint API mocking
- `mocks/test_data.go` - Test data generators

#### Sample Implementation
```go
// mocks/beacon_node_mock.go
//go:generate mockery --name=BeaconNodeInterface --output=. --filename=beacon_node_mock.go

type MockBeaconNode struct {
    mock.Mock
}

func (m *MockBeaconNode) GetBeaconBlock(ctx context.Context, identifier string) (*spec.VersionedSignedBeaconBlock, error) {
    args := m.Called(ctx, identifier)
    return args.Get(0).(*spec.VersionedSignedBeaconBlock), args.Error(1)
}

func (m *MockBeaconNode) Synced(ctx context.Context) (bool, error) {
    args := m.Called(ctx)
    return args.Bool(0), args.Error(1)
}

// Helper methods for common mock setups
func (m *MockBeaconNode) SetupSyncedResponse(synced bool) {
    m.On("Synced", mock.Anything).Return(synced, nil)
}

func (m *MockBeaconNode) SetupBlockResponse(identifier string, block *spec.VersionedSignedBeaconBlock) {
    m.On("GetBeaconBlock", mock.Anything, identifier).Return(block, nil)
}

// test_data.go - Test data generators
func GenerateTestBeaconBlock(slot uint64, parentRoot []byte) *spec.VersionedSignedBeaconBlock {
    return &spec.VersionedSignedBeaconBlock{
        Version: spec.DataVersionCapella,
        Capella: &capella.SignedBeaconBlock{
            Message: &capella.BeaconBlock{
                Slot:       phase0.Slot(slot),
                ParentRoot: parentRoot,
                // ... additional test data
            },
        },
    }
}
```

### 3. Core Component Unit Tests

#### Specific Changes
- Create comprehensive test suites for each major component
- Implement table-driven tests for complex logic
- Add edge case and error condition testing
- Create integration test helpers for component interactions

#### Files Affected
- `pkg/cannon/cannon_test.go` - Main component tests
- `pkg/cannon/config_test.go` - Configuration validation tests
- `pkg/cannon/ethereum/beacon_test.go` - Beacon node tests
- `pkg/cannon/deriver/` - Deriver component tests
- `pkg/cannon/iterator/` - Iterator logic tests

#### Sample Implementation
```go
// cannon_test.go
func TestCannon_Start(t *testing.T) {
    tests := []struct {
        name           string
        setupMocks     func(*MockBeaconNode, *MockCoordinator)
        expectedError  error
        validateState  func(*testing.T, *Cannon)
    }{
        {
            name: "successful_start_with_synced_beacon",
            setupMocks: func(beacon *MockBeaconNode, coord *MockCoordinator) {
                beacon.SetupSyncedResponse(true)
                beacon.On("GetSpecialForkSchedule").Return(&ForkSchedule{}, nil)
                coord.On("GetCannonLocation", mock.Anything, mock.Anything).Return(&xatu.GetCannonLocationResponse{
                    Location: &xatu.CannonLocation{
                        NetworkId: "mainnet",
                        Type:      "beacon_block",
                    },
                }, nil)
            },
            expectedError: nil,
            validateState: func(t *testing.T, c *Cannon) {
                assert.NotNil(t, c.beacon)
                assert.NotNil(t, c.forkSchedule)
            },
        },
        {
            name: "start_fails_with_unsynced_beacon",
            setupMocks: func(beacon *MockBeaconNode, coord *MockCoordinator) {
                beacon.SetupSyncedResponse(false)
            },
            expectedError: errors.New("beacon node not synced"),
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            mockBeacon := &MockBeaconNode{}
            mockCoordinator := &MockCoordinator{}
            tt.setupMocks(mockBeacon, mockCoordinator)
            
            cannon := &Cannon{
                beacon:      mockBeacon,
                coordinator: mockCoordinator,
                log:         logrus.NewEntry(logrus.New()),
            }
            
            // Execute
            err := cannon.Start(context.Background())
            
            // Verify
            if tt.expectedError != nil {
                assert.EqualError(t, err, tt.expectedError.Error())
            } else {
                assert.NoError(t, err)
                if tt.validateState != nil {
                    tt.validateState(t, cannon)
                }
            }
            
            mockBeacon.AssertExpectations(t)
            mockCoordinator.AssertExpectations(t)
        })
    }
}
```

### 4. Iterator and Deriver Component Tests

#### Specific Changes
- Test complex backfilling logic with various scenarios
- Validate epoch and slot calculation algorithms
- Test error handling and retry mechanisms
- Verify state persistence and recovery

#### Files Affected
- `pkg/cannon/iterator/backfilling_checkpoint_iterator_test.go`
- `pkg/cannon/deriver/beacon/eth/v1/beacon_block_test.go`
- `pkg/cannon/deriver/beacon/eth/v2/beacon_block_test.go`
- `pkg/cannon/deriver/blockprint/block_classification_test.go`

#### Sample Implementation
```go
// iterator/backfilling_checkpoint_iterator_test.go
func TestBackfillingCheckpointIterator_Next(t *testing.T) {
    tests := []struct {
        name              string
        currentLocation   *xatu.CannonLocation
        latestCheckpoint  *phase0.Checkpoint
        config           *Config
        expectedNext     *xatu.CannonLocation
        expectedBackfill bool
    }{
        {
            name: "advance_to_next_epoch_when_current_complete",
            currentLocation: &xatu.CannonLocation{
                Epoch:     100,
                Slot:      3199, // Last slot of epoch 100
                Type:      "beacon_block",
            },
            latestCheckpoint: &phase0.Checkpoint{
                Epoch: 105,
                Root:  [32]byte{1, 2, 3},
            },
            config: &Config{
                SlotsPerEpoch: 32,
            },
            expectedNext: &xatu.CannonLocation{
                Epoch: 101,
                Slot:  3200, // First slot of epoch 101
                Type:  "beacon_block",
            },
            expectedBackfill: false,
        },
        {
            name: "trigger_backfill_when_far_behind",
            currentLocation: &xatu.CannonLocation{
                Epoch: 100,
                Slot:  3200,
                Type:  "beacon_block",
            },
            latestCheckpoint: &phase0.Checkpoint{
                Epoch: 200, // 100 epochs ahead
                Root:  [32]byte{1, 2, 3},
            },
            config: &Config{
                SlotsPerEpoch:        32,
                BackfillThreshold:    50,
            },
            expectedBackfill: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mocks
            mockCoordinator := &MockCoordinator{}
            mockBeacon := &MockBeaconNode{}
            
            mockBeacon.On("GetSpecialForkSchedule").Return(&ForkSchedule{
                GenesisTime:   time.Now().Add(-time.Hour),
                SlotsPerEpoch: tt.config.SlotsPerEpoch,
            }, nil)
            
            mockCoordinator.On("GetCannonLocation", mock.Anything, mock.Anything).
                Return(&xatu.GetCannonLocationResponse{
                    Location: tt.currentLocation,
                }, nil)

            // Create iterator
            iterator := &BackfillingCheckpointIterator{
                beacon:      mockBeacon,
                coordinator: mockCoordinator,
                config:      tt.config,
                log:         logrus.NewEntry(logrus.New()),
            }

            // Execute
            next, shouldBackfill, err := iterator.Next(context.Background())

            // Verify
            assert.NoError(t, err)
            assert.Equal(t, tt.expectedBackfill, shouldBackfill)
            
            if tt.expectedNext != nil {
                assert.Equal(t, tt.expectedNext.Epoch, next.Epoch)
                assert.Equal(t, tt.expectedNext.Slot, next.Slot)
                assert.Equal(t, tt.expectedNext.Type, next.Type)
            }

            mockBeacon.AssertExpectations(t)
            mockCoordinator.AssertExpectations(t)
        })
    }
}
```

### 5. Configuration and Validation Tests

#### Specific Changes
- Test configuration validation logic
- Verify sink creation and initialization
- Test override application and precedence
- Validate error conditions and edge cases

#### Files Affected
- `pkg/cannon/config_test.go`
- Test files for each deriver configuration

#### Sample Implementation
```go
// config_test.go
func TestConfig_Validate(t *testing.T) {
    tests := []struct {
        name        string
        config      *Config
        expectedErr string
    }{
        {
            name: "valid_config_passes_validation",
            config: &Config{
                LoggingLevel: "info",
                MetricsAddr:  "localhost:9090",
                PProfAddr:    "localhost:6060",
                ProbeAddr:    "localhost:8080",
                Ethereum: ethereum.Config{
                    BeaconNodeAddress: "http://localhost:5052",
                },
                Coordinator: coordinator.Config{
                    Address: "localhost:8081",
                },
                Outputs: []output.Config{
                    {
                        Name: "stdout",
                        Type: "stdout",
                    },
                },
                Derivers: []deriver.Config{
                    {
                        Name:    "beacon_block",
                        Type:    "beacon_block_v2",
                        Enabled: true,
                    },
                },
            },
            expectedErr: "",
        },
        {
            name: "missing_beacon_node_address_fails",
            config: &Config{
                Ethereum: ethereum.Config{
                    BeaconNodeAddress: "",
                },
            },
            expectedErr: "beacon node address is required",
        },
        {
            name: "invalid_logging_level_fails",
            config: &Config{
                LoggingLevel: "invalid",
                Ethereum: ethereum.Config{
                    BeaconNodeAddress: "http://localhost:5052",
                },
            },
            expectedErr: "invalid logging level: invalid",
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

## Testing Strategy

### Unit Testing
- **Component isolation**: Each component tested in complete isolation using mocks
- **Interface boundaries**: All external dependencies mocked through interfaces
- **State management**: Complex state transitions tested with predictable inputs
- **Error conditions**: Comprehensive error path testing including network failures, timeouts, and invalid data
- **Edge cases**: Boundary conditions, empty inputs, and extreme values

### Integration Testing
- **Component interactions**: Test how components work together with real interfaces
- **Configuration flow**: End-to-end configuration validation and application
- **Lifecycle management**: Start/stop sequences and graceful shutdown
- **Mock integration**: Higher-level tests using multiple mocks working together

### Performance Testing
- **Memory usage**: Verify no memory leaks in long-running processes
- **Goroutine management**: Ensure proper cleanup of background workers
- **Resource utilization**: Test behavior under resource constraints

### Validation Criteria
- **Code coverage**: Minimum 90% line coverage, 85% branch coverage
- **Test execution time**: Complete suite runs in under 30 seconds
- **Test reliability**: All tests pass consistently in CI/CD environment
- **Mock verification**: All mock expectations verified in every test
- **Error coverage**: All error paths and edge cases tested

## Implementation Dependencies

### Phase 1: Foundation (Infrastructure Setup)
- [ ] Create interface abstractions for external dependencies
- [ ] Set up mock infrastructure using testify/mock
- [ ] Implement testable factory pattern for component creation
- [ ] Create test data generators and utilities
- [ ] Set up CI/CD test integration
- Dependencies: None (can start immediately)

### Phase 2: Core Component Testing (Main Logic)
- [ ] Implement Cannon main component tests
- [ ] Add configuration validation tests  
- [ ] Create BeaconNode wrapper tests
- [ ] Implement coordinator client tests
- [ ] Add blockprint client tests
- Dependencies: Phase 1 completion (interfaces and mocks)

### Phase 3: Complex Component Testing (Business Logic)
- [ ] Implement iterator logic tests (backfilling, checkpoint management)
- [ ] Add deriver component tests (all versions: v1, v2, blockprint)
- [ ] Create event processing pipeline tests
- [ ] Add metrics and monitoring tests
- Dependencies: Phase 2 completion (core components working)

### Phase 4: Integration and Validation (End-to-End)
- [ ] Implement integration test helpers
- [ ] Add performance and memory tests
- [ ] Create comprehensive end-to-end test scenarios
- [ ] Validate coverage targets and quality metrics
- [ ] Document testing patterns and best practices
- Dependencies: Phase 3 completion (all components tested)

## Risks and Considerations

### Implementation Risks
- **Complex refactoring required**: [Risk] Major structural changes needed for testability → [Mitigation] Implement interfaces gradually, maintain backward compatibility
- **External dependency mocking complexity**: [Risk] Complex external APIs difficult to mock accurately → [Mitigation] Create comprehensive mock scenarios, use contract testing
- **Time-dependent logic testing**: [Risk] Background processes and timing logic hard to test → [Mitigation] Use dependency injection for time/scheduling, controllable test clocks
- **State management complexity**: [Risk] Complex iterator and backfilling logic prone to test flakiness → [Mitigation] Focus on pure functions, deterministic state transitions

### Performance Considerations
- **Test execution speed**: [Performance concern] Large test suite may slow development workflow → [Addressing approach] Parallel test execution, fast mock implementations, selective test running
- **Mock overhead**: [Performance concern] Extensive mocking may impact test performance → [Addressing approach] Lightweight mock implementations, shared mock instances where appropriate

### Security Considerations
- **Credential exposure in tests**: [Security concern] Test credentials or keys accidentally committed → [Addressing approach] Use test-specific credentials, .gitignore test data, secret scanning
- **Mock data sensitivity**: [Security concern] Real network data used in tests → [Addressing approach] Generate synthetic test data, anonymize any real data used

### Quality Considerations
- **Test maintenance burden**: [Risk] Large test suite becomes maintenance overhead → [Mitigation] Clear testing patterns, shared utilities, regular test review and cleanup
- **Mock drift**: [Risk] Mocks diverge from real implementations → [Mitigation] Contract testing, regular mock validation against real services
- **Coverage metrics gaming**: [Risk] Focus on coverage numbers over quality → [Mitigation] Combine coverage with mutation testing, code review emphasis on test quality

## Expected Outcomes

### Immediate Outcomes
- **Complete unit test coverage**: All pkg/cannon components have comprehensive unit tests
- **Successful CI/CD integration**: `go test ./pkg/cannon/...` passes reliably in automated environments
- **Refactored architecture**: Components properly structured for testability with clear interfaces
- **Mock infrastructure**: Comprehensive mocking framework available for all external dependencies

### Long-term Outcomes  
- **Improved maintainability**: Confidence in making changes without breaking existing functionality
- **Faster development cycles**: Ability to quickly validate changes through comprehensive test suite
- **Better code quality**: Interface-driven design and dependency injection improve overall architecture
- **Reduced regression risk**: Strong test coverage prevents unintended behavioral changes

### Success Metrics
- **Line coverage**: Target >90% line coverage across all pkg/cannon components
- **Branch coverage**: Target >85% branch coverage for decision logic
- **Test execution time**: Complete test suite runs in <30 seconds
- **Test reliability**: 100% test success rate in CI/CD environment over 30-day period
- **Code quality**: Zero critical code smells related to testability in static analysis
- **Developer productivity**: Reduced time to validate changes (target: <5 minutes for full test run)