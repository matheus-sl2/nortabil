//go:build ignore

package main

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// BenchmarkResult armazena as métricas estatísticas coletadas durante o ensaio
type BenchmarkResult struct {
	TotalOperations int
	Concurrency     int
	TotalDuration   time.Duration
	Throughput      float64 // ops/segundo
	AvgLatency      time.Duration
	P50Latency      time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
}

func main() {
	fmt.Println("=================================================================")
	fmt.Println("   NORTABIL - PROTOCOLO DE TESTES DE CARGA E BENCHMARKING (TCC)   ")
	fmt.Println("   Motor de Partidas Dobradas com Validação Concorrente em Go    ")
	fmt.Println("=================================================================\n")

	scenarios := []struct {
		TotalOps    int
		Concurrency int
	}{
		{TotalOps: 10000, Concurrency: 10},
		{TotalOps: 50000, Concurrency: 50},
		{TotalOps: 100000, Concurrency: 100},
	}

	for _, sc := range scenarios {
		res := runBenchmarkScenario(sc.TotalOps, sc.Concurrency)
		printScenarioResult(res)
	}
}

func runBenchmarkScenario(totalOps, concurrency int) BenchmarkResult {
	latencies := make([]time.Duration, totalOps)
	opsPerWorker := totalOps / concurrency

	var wg sync.WaitGroup
	wg.Add(concurrency)

	accDebito := uuid.New()
	accCredito := uuid.New()
	amount := decimal.NewFromFloat(2450.85)

	startTime := time.Now()

	for w := 0; w < concurrency; w++ {
		workerID := w
		go func(wID int) {
			defer wg.Done()
			offset := wID * opsPerWorker
			limit := offset + opsPerWorker
			if wID == concurrency-1 {
				limit = totalOps // Garante resto da divisão
			}

			for i := offset; i < limit; i++ {
				opStart := time.Now()

				entry := domain.JournalEntry{
					ID:          uuid.New(),
					Date:        time.Now(),
					Description: "Lançamento contábil de estresse para validação de integridade",
					Lines: []domain.JournalLine{
						{
							AccountID: accDebito,
							Type:      domain.JournalLineDebit,
							Amount:    amount,
						},
						{
							AccountID: accCredito,
							Type:      domain.JournalLineCredit,
							Amount:    amount,
						},
					},
				}

				if err := entry.ValidateDoubleEntry(); err != nil {
					panic(err)
				}

				latencies[i] = time.Since(opStart)
			}
		}(workerID)
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	// Análise Estatística
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	var totalLat time.Duration
	for _, l := range latencies {
		totalLat += l
	}

	p50Idx := int(math.Round(float64(totalOps) * 0.50))
	p95Idx := int(math.Round(float64(totalOps) * 0.95))
	p99Idx := int(math.Round(float64(totalOps) * 0.99))

	if p50Idx >= totalOps {
		p50Idx = totalOps - 1
	}
	if p95Idx >= totalOps {
		p95Idx = totalOps - 1
	}
	if p99Idx >= totalOps {
		p99Idx = totalOps - 1
	}

	return BenchmarkResult{
		TotalOperations: totalOps,
		Concurrency:     concurrency,
		TotalDuration:   totalDuration,
		Throughput:      float64(totalOps) / totalDuration.Seconds(),
		AvgLatency:      totalLat / time.Duration(totalOps),
		P50Latency:      latencies[p50Idx],
		P95Latency:      latencies[p95Idx],
		P99Latency:      latencies[p99Idx],
		MinLatency:      latencies[0],
		MaxLatency:      latencies[totalOps-1],
	}
}

func printScenarioResult(r BenchmarkResult) {
	fmt.Printf("--- CENÁRIO: %d LANÇAMENTOS (%d GOROUTINES CONCORRENTES) ---\n", r.TotalOperations, r.Concurrency)
	fmt.Printf("Tempo Total Decorrido : %v\n", r.TotalDuration)
	fmt.Printf("Vazão (Throughput)    : %.2f lançamentos/segundo\n", r.Throughput)
	fmt.Printf("Latência Média        : %v\n", r.AvgLatency)
	fmt.Printf("Latência P50 (Mediana): %v\n", r.P50Latency)
	fmt.Printf("Latência P95          : %v\n", r.P95Latency)
	fmt.Printf("Latência P99          : %v\n", r.P99Latency)
	fmt.Printf("Latência Mínima       : %v\n", r.MinLatency)
	fmt.Printf("Latência Máxima       : %v\n\n", r.MaxLatency)
}
