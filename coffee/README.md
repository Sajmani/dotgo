# Introduction

The `coffee` program is a tool for studying parallelism, contention,
utilization, latency, and throughput.  It simulates making coffee with
a set of machines (grinder for beans, espresso maker, steamer for
milk) and measures the performance of various implementations.  The
measurements show the effects of various structural changes and
optimizations.

Run `coffee` with no arguments to simulate the "ideal" scenario with 1
worker.  Run `coffee --help` to see flags for configuring the
simulation.  See `generate.sh` for commands that generate measurements
for many different scenarios.

# Notes

Run `generate.sh` to regenerate `*.csv` files.

Use `taskset -c 0-5 go run *.go` to restrict execution to specified CPU IDs.

Use https://github.com/aclements/perflock to prevent multiple benchmarks from
running at once and keep CPU from running too hot (and so triggering CPU
throttling).

# Generate torch graphs

```shell
foreach mode ( \
  ideal locking finelocking parsteam americano espresso \
  multi-1 multi-2 multi-4 multi-8 \
  linearpipe-0 linearpipe-1 linearpipe-10 \
  splitpipe-0 splitpipe-1 splitpipe-10 \
  multipipe-1 multipipe-2 multipipe-4 multipipe-8 \
  )
taskset -c 0-5 go run *.go --dur=3s --par=0 --trace=./trace-$mode.out -mode=$mode
go tool trace --pprof=sync ./trace-$mode.out > ./sync-$mode.pprof
go-torch -b ./sync-$mode.pprof -f ./torch-$mode.svg
end
```
