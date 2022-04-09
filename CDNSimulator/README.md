
# C2DN simulator 

## dependency (Using Ubuntu as an example)
#### Install dependencies from apt-get (step 1)
```
sudo apt install -yqq make libglib2.0-dev libgoogle-perftools-dev libboost-all-dev 
```

#### Install cmake (step 2)
```
wget https://github.com/Kitware/CMake/releases/download/v3.23.0/cmake-3.23.0-linux-x86_64.sh
bash cmake-3.23.0-linux-x86_64.sh
```

#### Install OR-tools (step 3)
```
sudo apt-get -y install pkg-config build-essential autoconf libtool zlib1g-dev lsb-release
git clone https://github.com/google/or-tools
cd or-tools
mkdir _build
cd _build
cmake -DBUILD_DEPS=ON ..
make -j
sudo make install
```


## build CDNSimulator
```

mkdir _build; 
cd _build; 
cmake ..; 
make -j; 

```

## run the simulator 
```
  -h [ --help ]                      Print help messages
  --name arg (=cluster)              exp name
  -a [ --alg ] arg                   cache replacement algorithm
  -d [ --dataPath ] arg              data path
  -s [ --serverCacheSize ] arg       per server cache size is each server has the same size
  --serverCacheSizes arg             list of cache sizes if servers have different sizes
  -m [ --nServer ] arg               the number of cache servers
  -t [ --traceFormat ] arg           the format of trace
  -l [ --logInterval ] arg           the log output interval in virtual time
  -n [ --EC_n ] arg                  N in erasure coding (the total number of chunks) 
  -k [ --EC_k ] arg                  K in erasure coding (the number of data chunks)
  -o [ --admission ] arg (=0)        n-hit-wonder filters 
  -z [ --EC_sizeThreshold ] arg (=0) size threshold for coding 
  --checkOneMore arg                 whether check one more server
  -b [ --rebalance ] arg             whether use parity to rebalance write
  --clusterMode arg                  mode of cluster, support
                                     two_rep_popularity, two_rep_always, no_rep, C2DN
  -g [ --gutterSpace ] arg           whether use gutter space
  -f [ --failureData ] arg           Path to failure data
```

Example configuration
```
./simulator --alg lrusize --dataPath DATA --traceType akamai1b --nServer 10 --cacheSize 2000000000 --n 2 --k 1 --x 0 -z 128000 --admission 0 --logInterval 20000
```

Example input
* The request trace is parsed by libCacheSim and it supports several format, see the repo of libCacheSim for the format
* The unavailabilily trace has N lines, where line i is the index of the servers that are unavailable in the ith 5-minute window, when there is no unavailability, just leave it as an empty line. 



## License
```
Copyright 2022, Carnegie Mellon University

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

## Support 
This work was supported in part by Facebook Fellowship, NSF grants CNS 1901410, CNS 1956271, CNS 1763617, and a AWS grant.
