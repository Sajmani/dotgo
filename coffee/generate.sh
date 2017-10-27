#!/bin/zsh

echo vary CPU

echo "" > varycpu.csv
foreach i ( 0 1 2 3 4 5 )
taskset -c 0-$i go run *.go --header=false --par=0 --dur=4s --mode=ideal,locking,finelocking,parsteam,americano,espresso,linearpipe-0,linearpipe-1,linearpipe-10,splitpipe-0,splitpipe-1,americanopipe-0,americanopipe-1,espressopipe-0,espressopipe-1,multi-1,multi-2,multi-4,multipipe-1,multipipe-2,multipipe-4 2> /dev/null >> varycpu.csv
end

echo overload

taskset -c 0-5 go run *.go --par=10,100,1000,10000 --dur=4s --mode=ideal,locking,finelocking,parsteam,americano,espresso,linearpipe-0,linearpipe-1,linearpipe-10,splitpipe-0,splitpipe-1,americanopipe-0,americanopipe-1,espressopipe-0,espressopipe-1,multi-1,multi-2,multi-4,multipipe-1,multipipe-2,multipipe-4 2> /dev/null > overload.csv

echo jitter

taskset -c 0-5 go run *.go --par=0 --jitter=1ms --dur=4s --mode=ideal,locking,finelocking,parsteam,americano,espresso,linearpipe-0,linearpipe-1,linearpipe-10,splitpipe-0,splitpipe-1,americanopipe-0,americanopipe-1,espressopipe-0,espressopipe-1,multi-1,multi-2,multi-4,multipipe-1,multipipe-2,multipipe-4 2> /dev/null > jitter.csv

echo done
