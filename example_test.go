package zero2algo

import (
	"context"
	"encoding/csv"
	"os"
	"sync"

	"github.com/colngroup/zero2algo/broker/backtest"
	"github.com/colngroup/zero2algo/market"
	"github.com/colngroup/zero2algo/optimize"
	"github.com/colngroup/zero2algo/perf"
	"github.com/colngroup/zero2algo/trader/hodl"
)

func Example() {
	// Verbose error handling ommitted for brevity

	// Identify the asset to trade
	//asset := market.NewAsset("BTCUSD")

	// Define the set of possible values for each param
	params := map[string]any{
		hodl.BuyBarIndexKey:  []any{0, 1, 1000},
		hodl.SellBarIndexKey: []any{0, 1000, 2000},
	}
	// Build a set of test cases, one for each permutation of params
	cases := optimize.CartesianBuilder(params)

	// Slice to store each report created by execution of a test case
	results := make([]perf.PerformanceReport, 0, len(cases))

	// Read a .csv file of historical prices (aka candlestick data)
	// Cache the prices in memory to use in multiple optimization iterations
	file, _ := os.Open("example_prices.csv")
	defer file.Close()
	prices, _ := market.NewCSVKlineReader(csv.NewReader(file)).ReadAll()

	// Iterate the test cases, executing each set of params and collecting the results
	// Test cases are executed concurrently to reduce run time
	wg := new(sync.WaitGroup)
	for _, c := range cases {
		wg.Add(1)

		go func(c map[string]any) {
			defer wg.Done()

			// Create a special simulated dealer for each test case run
			dealer := backtest.NewDealer()

			// Create a new bot initialized with our dealer
			// The bot is configured with the params in the test case
			// Hodl Bot implements a basic buy and hold algo
			bot, _ := hodl.MakeBotFromConfig(c)
			bot.SetDealer(dealer)

			// Iterate prices sending each price interval to the dealer and then to the bot
			for _, price := range prices {
				dealer.ReceivePrice(context.Background(), price)
				bot.ReceivePrice(context.Background(), price)
			}
			// Close the bot which will liquidate any open position resulting in a final trade
			bot.Close(context.Background())

			// Generate a performance report for the test case and add it to the result set
			trades, _, _ := dealer.ListTrades(context.Background(), nil)
			equity := dealer.EquityHistory()
			results = append(results, perf.NewPerformanceReport(trades, equity))
		}(c)
	}
	wg.Wait()

	// Rank results based on the test case with the highest sharpe ratio
	//slices.SortFunc(results, optimize.SharpeRanker)
	perf.PrintSummary(results[len(results)-1])

}
