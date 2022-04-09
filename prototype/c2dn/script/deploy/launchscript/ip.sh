#!/bin/bash 


export cdn_pub_ips=()
export cdn_priv_ips=()
export origin_ip=0.0.0.0
export client_ip=127.0.0.1


if [[ ${expname} == cloudlab_CDN_akamai1_expLatency_100G ]]; then
    # export origin_ip=ms0133.lab.onelab.eu
    # export cdn_pub_ips=(ms1343.utah.cloudlab.us ms1319.utah.cloudlab.us ms1333.utah.cloudlab.us ms1337.utah.cloudlab.us ms1331.utah.cloudlab.us ms1310.utah.cloudlab.us ms1342.utah.cloudlab.us ms1307.utah.cloudlab.us ms1325.utah.cloudlab.us ms1338.utah.cloudlab.us)
    # export cdn_priv_ips=(10.10.1.1 10.10.1.2 10.10.1.3 10.10.1.4 10.10.1.5 10.10.1.6 10.10.1.7 10.10.1.8 10.10.1.9 10.10.1.10)
    # export client_ip=clnode103.clemson.cloudlab.us
    # export CLOUDLAB_NFS_SERVER=ms1318.utah.cloudlab.us
    echo 

elif [[ ${expname} == cloudlab_C2DN_akamai1_expLatency_431_100G ]]; then
    echo 

elif [[ ${expname} == cloudlab_debug_akamai1Unavailability_expUnavailability_431_100G ]]; then
    echo 


elif [[ ${expname} == cloudlab_CDN_expUn*_akamai* ]]; then
    echo

elif [[ ${expname} == cloudlab_C2DN_expUn*_akamai* ]]; then
    echo



elif [[ ${expname} == aws_CDN_akamai1_*_100G ]]; then
    export origin_ip=3.124.217.103
    export cdn_pub_ips=(3.220.164.216 35.170.57.64 3.237.77.10 54.236.32.32 3.235.98.32 3.92.74.44 3.236.202.59 3.236.38.208 18.209.93.14 3.239.244.0)
    export cdn_priv_ips=(172.31.75.132 172.31.78.141 172.31.79.222 172.31.77.240 172.31.64.130 172.31.64.166 172.31.68.223 172.31.76.217 172.31.64.51 172.31.71.32)
    export client_ip=99.79.57.233

elif [[ ${expname} == aws_C2DN_akamai1_*_43_100G ]]; then
    export origin_ip=18.156.3.48
    export cdn_pub_ips=(3.224.127.68 3.237.174.13 3.236.130.1 3.238.64.219 35.153.49.255 3.239.229.129 3.236.41.54 3.226.235.53 54.174.181.189 52.202.227.64)
    export cdn_priv_ips=(172.31.77.162 172.31.64.197 172.31.67.65 172.31.74.234 172.31.74.187 172.31.68.238 172.31.73.64 172.31.68.94 172.31.64.232 172.31.69.119)
    export client_ip=3.96.53.157


elif [[ ${expname} == aws_CDN_akamai2_*_1000G ]] || [[ "${expname}" == aws_noRep_akamai2* ]] ; then
    export origin_ip=3.124.217.103  
    export cdn_pub_ips=(3.220.164.216 35.170.57.64 3.237.77.10 54.236.32.32 3.235.98.32 3.92.74.44 3.236.202.59 3.236.38.208 18.209.93.14 3.239.244.0)
    export cdn_priv_ips=(172.31.75.132 172.31.78.141 172.31.79.222 172.31.77.240 172.31.64.130 172.31.64.166 172.31.68.223 172.31.76.217 172.31.64.51 172.31.71.32)
    export client_ip=99.79.57.233


elif [[ ${expname} == aws_C2DN_akamai2_*_43_1000G ]] || [[ "${expname}" == aws_naiveCoding_akamai2* ]]; then
    export origin_ip=18.156.3.48
    export cdn_pub_ips=(3.224.127.68 3.237.174.13 3.236.130.1 3.238.64.219 35.153.49.255 3.239.229.129 3.236.41.54 3.226.235.53 54.174.181.189 52.202.227.64)
    export cdn_priv_ips=(172.31.77.162 172.31.64.197 172.31.67.65 172.31.74.234 172.31.74.187 172.31.68.238 172.31.73.64 172.31.68.94 172.31.64.232 172.31.69.119)
    export client_ip=3.96.53.157


elif [[ ${expname} == aws_CDN_akamai2Unavailability_expUnavailability_3000G ]]; then
    # export origin_ip=52.58.16.214
    echo


elif [[ ${expname} == aws_C2DN_akamai2_expLatency_431_3000G ]]; then
    # export origin_ip=3.122.246.99
    echo

elif [[ "${expname}" == *"test"* ]]; then
    export origin_ip=18.192.176.62
    export cdn_pub_ips=(3.221.170.57 3.235.2.243 34.201.6.55 34.234.167.230 3.239.254.123 3.238.56.72 34.200.229.202 3.236.43.57 3.232.95.245 3.236.153.98)
    export cdn_priv_ips=(172.31.76.36 172.31.71.206 172.31.76.68 172.31.79.27 172.31.70.240 172.31.71.116 172.31.79.161 172.31.76.225 172.31.79.145 172.31.75.8)


else
    echo unknown expname ${expname}
fi

