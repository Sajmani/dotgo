#!/bin/zsh

echo vary CPU

echo "" > varycpu.csv
foreach i ( 0 1 2 3 4 5 )
taskset -c 0-$i go run *.go --header=false --par=0 --dur=10s --mode=ideal,locking,finelocking,parsteam,americano,espresso,linearpipe-0,linearpipe-1,linearpipe-10,splitpipe-0,splitpipe-1,americanopipe-0,americanopipe-1,espressopipe-0,espressopipe-1,multi-1,multi-2,multi-4,multipipe-1,multipipe-2,multipipe-4 2> /dev/null >> varycpu.csv
end

echo overload

taskset -c 0-5 go run *.go --par=10,100,1000,10000 --dur=10s --mode=ideal,locking,finelocking,parsteam,americano,espresso,linearpipe-0,linearpipe-1,linearpipe-10,splitpipe-0,splitpipe-1,americanopipe-0,americanopipe-1,espressopipe-0,espressopipe-1,multi-1,multi-2,multi-4,multipipe-1,multipipe-2,multipipe-4 2> /dev/null > overload.csv

echo multi

taskset -c 0-5 go run *.go --par=10,100,1000,10000 --dur=10s --mode=multi-1,multi-2,multi-3,multi-4,multi-5,multi-6,multi-7,multi-8,multi-9,multi-10 2> /dev/null > multi.csv

echo jitter

taskset -c 0-5 go run *.go --par=0 --jitter=1ms --dur=10s --mode=ideal,locking,finelocking,parsteam,americano,espresso,linearpipe-0,linearpipe-1,linearpipe-10,splitpipe-0,splitpipe-1,americanopipe-0,americanopipe-1,espressopipe-0,espressopipe-1,multi-1,multi-2,multi-4,multipipe-1,multipipe-2,multipipe-4 2> /dev/null > jitter.csv

echo done
