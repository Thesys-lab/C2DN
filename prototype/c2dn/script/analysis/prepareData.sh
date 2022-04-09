#!/bin/bash

set -e
aws s3 sync s3://juncheng-data/C2DN/prototype/2021-01-29/ .

################## prepare data #################
for f in *; do
    pushd $f; 
    if [ -d cdn0 ]; then popd; continue; fi
    for i in `seq 0 9`; do 
        (unzip -q c2dn_cdn_${i}.zip -d cdn${i}; sleep 0.2; mv cdn${i}/tmp/* cdn${i}/; sleep 0.2; rmdir cdn${i}/tmp/) &
    done
    unzip -q c2dn_client.zip; mv tmp client 
    unzip -q c2dn_origin.zip; mv tmp origin 
    wait
    cd client/c2dn/output/; 
    cat client.latency.firstByte.RAM client.latency.firstByte.Hit client.latency.firstByte.Miss > client.latency.firstByte.all; sort -n -k 1 client.latency.firstByte.all > client.latency.firstByte.all.sort
    cat client.latency.fullResp.RAM client.latency.fullResp.Hit client.latency.fullResp.Miss > client.latency.fullResp.all; sort -n -k 1 client.latency.fullResp.all > client.latency.fullResp.all.sort
    popd;
done


################# analyze sys level I/O ################### 
rm io 2>/dev/null || true
for i in `seq 0 9`; do 
    grep nvme0n1 cdn${i}/c2dn/stat/host | awk 'NR%2{r=$4;w=$8;next}{print ($4-r, $8-w)}' >> io
    grep nvme1n1 cdn${i}/c2dn/stat/host | awk 'NR%2{r=$4;w=$8;next}{print ($4-r, $8-w)}' >> io
done
echo -e "
TB=1000*1000*1000
import numpy as np
np.set_printoptions(precision=2)
read, write=[0]*10, [0]*10
f1=open('io')
for i, line in enumerate(f1): 
  ls=line.strip().split()
  read[i//2%10] += int(ls[0])
  write[i//2%10] += int(ls[1])
print(np.array(read)/TB, sum(read)/TB)
print(np.array(write)/TB, sum(write)/TB)
" > an.py
python3 an.py


cd ../../../
cd cdn6/c2dn/output; 
cat client.latency.firstByte.Hit client.latency.firstByte.RAM client.latency.firstByte.Miss > client.latency.firstByte.all; sort -n -k 1 client.latency.firstByte.all > client.latency.firstByte.all.sort

for i in `seq 0 9`; do 
    grep -A 30 "all Origins combined" cdn${i}/c2dn/ats_stat | grep "Cache hit total" 
done

for i in `seq 0 9`; do 
    grep -A 30 "all Origins combined" cdn${i}/c2dn/ats_stat | grep "Cache miss total" 
done


for i in `seq 0 9`; do 
    grep '"origin"' cdn${i}/c2dn/metricFE
    grep 'allToClient' cdn${i}/c2dn/metricFE
    grep 'intra' cdn${i}/c2dn/metricFE
done


grep allToClient $f
