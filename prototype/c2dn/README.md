
# C2DN prototype 

## Installing dependency
Install golang
```
wget https://go.dev/dl/go1.18.linux-amd64.tar.gz
tar xvf go1.18.linux-amd64.tar.gz
sudo mv go /usr/local/
echo 'PATH=$PATH:/usr/local/go/bin/' >> $HOME/.bashrc 
source $HOME/.bashrc
```




## Build 
```

go build ./cmd/frontend; 
go build ./cmd/client; 
go build ./cmd/origin; 

```

## Run the prototype 
Install and start Apache traffic server on each edge cluster 


Run the origin server 
```
./origin
```

Run the client 
```
./client 
```


Run the CDN edge cluster 
```
./frontend -mode="twoRepAlways" -EC_n=2 -EC_k=1 -ramCacheSize=128000000 -nServers=10 -nodeIdx=2 [list of ATS ip:port]

go build ./cmd/frontend; ./frontend -mode=C2DN -EC_n=4 -EC_k=3 -ramCacheSize=128000000 -nServers=10 -nodeIdx=2 [list of ATS ip:port]
```


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
