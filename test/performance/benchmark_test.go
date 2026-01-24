package performance

import (
	"encoding/json"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/carlosarraes/bt/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Performance targets for the bt CLI
const (
	MaxStartupTime     = 100 * time.Millisecond // Command startup time
	MaxAPIResponseTime = 500 * time.Millisecond // API response time
	MaxMemoryUsage     = 50 * 1024 * 1024       // 50MB memory usage
	MaxBinarySize      = 20 * 1024 * 1024       // 20MB binary size
	MinCommandsPerSec  = 10                     // Minimum commands per second
)

// BenchmarkCommandStartup benchmarks command startup time
func BenchmarkCommandStartup(b *testing.B) {
	binaryPath := utils.BuildBinary(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := utils.RunCommand(binaryPath, "version")
		if result.ExitCode != 0 {
			b.Fatalf("Command failed: %s", result.Stderr)
		}
	}
}

// BenchmarkVersionCommand benchmarks the version command specifically
func BenchmarkVersionCommand(b *testing.B) {
	binaryPath := utils.BuildBinary(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		result := utils.RunCommand(binaryPath, "version")
		duration := time.Since(start)

		if result.ExitCode != 0 {
			b.Fatalf("Version command failed: %s", result.Stderr)
		}

		if duration > MaxStartupTime {
			b.Logf("Warning: Command took %v, expected < %v", duration, MaxStartupTime)
		}
	}
}

// BenchmarkHelpCommand benchmarks the help command
func BenchmarkHelpCommand(b *testing.B) {
	binaryPath := utils.BuildBinary(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := utils.RunCommand(binaryPath, "--help")
		if result.ExitCode != 0 {
			b.Fatalf("Help command failed: %s", result.Stderr)
		}
	}
}

// BenchmarkJSONOutput benchmarks JSON output formatting
func BenchmarkJSONOutput(b *testing.B) {
	binaryPath := utils.BuildBinary(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := utils.RunCommand(binaryPath, "--output", "json", "version")
		if result.ExitCode != 0 {
			b.Fatalf("JSON output command failed: %s", result.Stderr)
		}

		// Verify it's valid JSON
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(result.Stdout), &jsonData); err != nil {
			b.Fatalf("Invalid JSON output: %v", err)
		}
	}
}

// BenchmarkConcurrentCommands benchmarks concurrent command execution
func BenchmarkConcurrentCommands(b *testing.B) {
	binaryPath := utils.BuildBinary(b)
	const numConcurrent = 10

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		start := time.Now()

		for j := 0; j < numConcurrent; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				result := utils.RunCommand(binaryPath, "version")
				if result.ExitCode != 0 {
					b.Errorf("Concurrent command failed: %s", result.Stderr)
				}
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		commandsPerSec := float64(numConcurrent) / duration.Seconds()
		if commandsPerSec < MinCommandsPerSec {
			b.Logf("Warning: Commands per second %.2f, expected >= %d", commandsPerSec, MinCommandsPerSec)
		}
	}
}

// BenchmarkAPIClient benchmarks API client performance (if integration tests enabled)
func BenchmarkAPIClient(b *testing.B) {
	if !utils.IsIntegrationTest() {
		b.Skip("Skipping API benchmark. Set BT_INTEGRATION_TESTS=1 to run.")
	}

	client := utils.NewHTTPClient()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		resp, err := client.Get("/user")
		duration := time.Since(start)

		if err != nil {
			b.Fatalf("API request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 && resp.StatusCode != 429 { // Allow rate limiting
			b.Fatalf("API request failed with status %d", resp.StatusCode)
		}

		if duration > MaxAPIResponseTime {
			b.Logf("Warning: API request took %v, expected < %v", duration, MaxAPIResponseTime)
		}

		// Add small delay to avoid rate limiting
		time.Sleep(10 * time.Millisecond)
	}
}

// BenchmarkMemoryUsage benchmarks memory usage during command execution
func BenchmarkMemoryUsage(b *testing.B) {
	binaryPath := utils.BuildBinary(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var memBefore, memAfter runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memBefore)

		result := utils.RunCommand(binaryPath, "version")
		if result.ExitCode != 0 {
			b.Fatalf("Command failed: %s", result.Stderr)
		}

		runtime.GC()
		runtime.ReadMemStats(&memAfter)

		memUsed := memAfter.Alloc - memBefore.Alloc
		if memUsed > MaxMemoryUsage {
			b.Logf("Warning: Memory usage %d bytes, expected < %d bytes", memUsed, MaxMemoryUsage)
		}
	}
}

// BenchmarkConfigParsing benchmarks configuration file parsing
func BenchmarkConfigParsing(b *testing.B) {
	testCtx := utils.NewTestContext(b)
	defer testCtx.Close()

	binaryPath := utils.BuildBinary(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := utils.RunCommand(binaryPath, "--config", testCtx.ConfigFile, "version")
		if result.ExitCode != 0 {
			b.Fatalf("Config parsing failed: %s", result.Stderr)
		}
	}
}

// TestPerformanceTargets tests that performance targets are met
func TestPerformanceTargets(t *testing.T) {
	binaryPath := utils.BuildBinary(t)

	t.Run("StartupTime", func(t *testing.T) {
		start := time.Now()
		result := utils.RunCommand(binaryPath, "version")
		duration := time.Since(start)

		utils.AssertCommandSuccess(t, result)
		assert.Less(t, duration, MaxStartupTime,
			"Command startup time should be less than %v, got %v", MaxStartupTime, duration)
	})

	t.Run("BinarySize", func(t *testing.T) {
		fileInfo, err := os.Stat(binaryPath)
		require.NoError(t, err)

		size := fileInfo.Size()
		assert.Less(t, size, int64(MaxBinarySize),
			"Binary size should be less than %d bytes, got %d bytes", MaxBinarySize, size)
	})

	t.Run("MemoryUsage", func(t *testing.T) {
		var memBefore, memAfter runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memBefore)

		result := utils.RunCommand(binaryPath, "version")
		utils.AssertCommandSuccess(t, result)

		runtime.GC()
		runtime.ReadMemStats(&memAfter)

		memUsed := memAfter.Alloc - memBefore.Alloc
		assert.Less(t, memUsed, uint64(MaxMemoryUsage),
			"Memory usage should be less than %d bytes, got %d bytes", MaxMemoryUsage, memUsed)
	})
}

// TestAPIPerformance tests API performance targets
func TestAPIPerformance(t *testing.T) {
	if !utils.IsIntegrationTest() {
		t.Skip("Skipping API performance test. Set BT_INTEGRATION_TESTS=1 to run.")
	}

	client := utils.NewHTTPClient()

	t.Run("ResponseTime", func(t *testing.T) {
		start := time.Now()
		resp, err := client.Get("/user")
		duration := time.Since(start)

		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode == 429 {
			t.Skip("Rate limited, skipping performance test")
		}

		require.Less(t, resp.StatusCode, 400, "API request should succeed")

		assert.Less(t, duration, MaxAPIResponseTime,
			"API response time should be less than %v, got %v", MaxAPIResponseTime, duration)
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		const numConcurrent = 5
		var wg sync.WaitGroup
		results := make([]time.Duration, numConcurrent)

		start := time.Now()

		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				reqStart := time.Now()
				resp, err := client.Get("/user")
				results[index] = time.Since(reqStart)

				if err != nil {
					t.Errorf("Concurrent request failed: %v", err)
					return
				}
				defer resp.Body.Close()

				// Skip rate limit responses
				if resp.StatusCode == 429 {
					return
				}

				if resp.StatusCode >= 400 {
					t.Errorf("Concurrent request failed with status %d", resp.StatusCode)
				}
			}(i)
		}

		wg.Wait()
		totalDuration := time.Since(start)

		// Check individual request times
		for i, duration := range results {
			if duration > MaxAPIResponseTime {
				t.Logf("Warning: Concurrent request %d took %v, expected < %v", i, duration, MaxAPIResponseTime)
			}
		}

		// Check overall throughput
		requestsPerSec := float64(numConcurrent) / totalDuration.Seconds()
		t.Logf("Concurrent requests per second: %.2f", requestsPerSec)
	})
}

// TestCommandThroughput tests command execution throughput
func TestCommandThroughput(t *testing.T) {
	binaryPath := utils.BuildBinary(t)
	const numCommands = 20
	const maxDuration = 5 * time.Second

	start := time.Now()

	for i := 0; i < numCommands; i++ {
		result := utils.RunCommand(binaryPath, "version")
		if result.ExitCode != 0 {
			t.Fatalf("Command %d failed: %s", i, result.Stderr)
		}
	}

	duration := time.Since(start)
	commandsPerSec := float64(numCommands) / duration.Seconds()

	assert.Less(t, duration, maxDuration,
		"Should execute %d commands in less than %v, took %v", numCommands, maxDuration, duration)

	assert.GreaterOrEqual(t, commandsPerSec, float64(MinCommandsPerSec),
		"Should execute at least %d commands per second, got %.2f", MinCommandsPerSec, commandsPerSec)

	t.Logf("Command throughput: %.2f commands/second", commandsPerSec)
}

// TestResourceUsage tests resource usage under load
func TestResourceUsage(t *testing.T) {
	binaryPath := utils.BuildBinary(t)
	const numIterations = 50

	var maxMemory uint64
	var totalDuration time.Duration

	for i := 0; i < numIterations; i++ {
		var memBefore, memAfter runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memBefore)

		start := time.Now()
		result := utils.RunCommand(binaryPath, "version")
		duration := time.Since(start)

		utils.AssertCommandSuccess(t, result)

		runtime.GC()
		runtime.ReadMemStats(&memAfter)

		memUsed := memAfter.Alloc - memBefore.Alloc
		if memUsed > maxMemory {
			maxMemory = memUsed
		}

		totalDuration += duration
	}

	avgDuration := totalDuration / numIterations

	assert.Less(t, maxMemory, uint64(MaxMemoryUsage),
		"Maximum memory usage should be less than %d bytes, got %d bytes", MaxMemoryUsage, maxMemory)

	assert.Less(t, avgDuration, MaxStartupTime,
		"Average startup time should be less than %v, got %v", MaxStartupTime, avgDuration)

	t.Logf("Resource usage over %d iterations:", numIterations)
	t.Logf("  Max memory: %d bytes", maxMemory)
	t.Logf("  Average duration: %v", avgDuration)
}

// TestColdStart tests cold start performance
func TestColdStart(t *testing.T) {
	binaryPath := utils.BuildBinary(t)

	// Measure first execution (cold start)
	start := time.Now()
	result := utils.RunCommand(binaryPath, "version")
	coldStartDuration := time.Since(start)

	utils.AssertCommandSuccess(t, result)

	// Measure subsequent execution (warm start)
	start = time.Now()
	result = utils.RunCommand(binaryPath, "version")
	warmStartDuration := time.Since(start)

	utils.AssertCommandSuccess(t, result)

	assert.Less(t, coldStartDuration, MaxStartupTime*2,
		"Cold start should be less than %v, got %v", MaxStartupTime*2, coldStartDuration)

	assert.Less(t, warmStartDuration, MaxStartupTime,
		"Warm start should be less than %v, got %v", MaxStartupTime, warmStartDuration)

	t.Logf("Cold start: %v, Warm start: %v", coldStartDuration, warmStartDuration)
}

// TestLargeOutput tests performance with large output
func TestLargeOutput(t *testing.T) {
	binaryPath := utils.BuildBinary(t)

	// Help command typically produces larger output
	start := time.Now()
	result := utils.RunCommand(binaryPath, "--help")
	duration := time.Since(start)

	utils.AssertCommandSuccess(t, result)

	outputSize := len(result.Stdout)
	assert.Greater(t, outputSize, 100, "Help output should be substantial")

	assert.Less(t, duration, MaxStartupTime*2,
		"Large output command should complete in less than %v, got %v", MaxStartupTime*2, duration)

	t.Logf("Large output test: %d bytes in %v", outputSize, duration)
}

// Additional benchmarks for specific scenarios

// BenchmarkErrorHandling benchmarks error handling performance
func BenchmarkErrorHandling(b *testing.B) {
	binaryPath := utils.BuildBinary(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use an invalid command to trigger error handling
		result := utils.RunCommand(binaryPath, "invalid-command")
		if result.ExitCode == 0 {
			b.Fatalf("Expected error command to fail")
		}
	}
}

// BenchmarkFlagParsing benchmarks flag parsing performance
func BenchmarkFlagParsing(b *testing.B) {
	binaryPath := utils.BuildBinary(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := utils.RunCommand(binaryPath, "--verbose", "--output", "json", "--no-color", "version")
		if result.ExitCode != 0 {
			b.Fatalf("Flag parsing failed: %s", result.Stderr)
		}
	}
}

// Helper functions for benchmarking

// measureTime measures execution time of a function
func measureTime(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// measureMemory measures memory usage of a function
func measureMemory(fn func()) uint64 {
	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	fn()

	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	return memAfter.Alloc - memBefore.Alloc
}
