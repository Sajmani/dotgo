foreach mode ( ideal locking finelocking multi-2 multi-4 multi-8 parsteam linearpipe-0 linearpipe-1 linearpipe-10 splitpipe-0 splitpipe-1 splitpipe-10 )
go run main.go burn.go perf.go --dur=1s --par=8 --maxq=0 --trace=./trace-$mode.out -mode=$mode
go tool trace --pprof=sync ./trace-$mode.out > ./sync-$mode.pprof
go-torch -b ./sync-$mode.pprof -f ./torch-$mode.svg
end
