# Introduction

TODO(sameer): write me :-)


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

# Disable hyperthreading on 12-core Linux machine

```shell
sudo su root
echo 0 > /sys/devices/system/cpu/cpu6/online
echo 0 > /sys/devices/system/cpu/cpu7/online
echo 0 > /sys/devices/system/cpu/cpu8/online
echo 0 > /sys/devices/system/cpu/cpu9/online
echo 0 > /sys/devices/system/cpu/cpu10/online
echo 0 > /sys/devices/system/cpu/cpu11/online
```

Better, use `taskset -c 0-5 go run *.go` to restrict execution to specified CPU IDs.

# Notes

Consider using https://github.com/aclements/perflock to prevent multiple
benchmarks from running at once and keep CPU from running too hot (and so
triggering CPU throttling).

# Generate data for 1-6 CPUs

```shell
echo "" > varycpu.csv
foreach i ( 0 1 2 3 4 5 )
taskset -c 0-$i go run *.go --par=0 --dur=4s --header=false --mode=ideal,locking,finelocking,parsteam,americano,espresso,linearpipe-0,linearpipe-1,linearpipe-10,splitpipe-0,splitpipe-1,americanopipe-0,americanopipe-1,espressopipe-0,espressopipe-1,multi-1,multi-2,multi-4,multipipe-1,multipipe-2,multipipe-4 2> /dev/null >> varycpu.csv
end
```

# Generate data for 6 CPUs with 10-10K users

```shell
taskset -c 0-5 go run *.go --par=10,100,1000,10000 --dur=4s  --header=false --mode=ideal,locking,finelocking,parsteam,americano,espresso,linearpipe-0,linearpipe-1,linearpipe-10,splitpipe-0,splitpipe-1,americanopipe-0,americanopipe-1,espressopipe-0,espressopipe-1,multi-1,multi-2,multi-4,multipipe-1,multipipe-2,multipipe-4 2> /dev/null > overload.csv
```
