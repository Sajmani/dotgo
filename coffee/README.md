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

Use https://github.com/aclements/perflock to prevent multiple benchmarks from
running at once and keep CPU from running too hot (and so triggering CPU
throttling).

Run `generate.sh` to regenerate csv files.
