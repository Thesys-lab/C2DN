#!/bin/bash


import() {
    local curr_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
    source "${curr_dir}/const.sh"
    source "${launch_script_dir}/params.sh"
    [ -z ${origin_ip+x} ] && source "${launch_script_dir}/ip.sh" || true
}

import


checkResponse() {
    url=$1
    correct_resp=$2

    resp=$(curl -s ${url[@]} || true)
    if [[ "$resp" != *"${correct_resp}"* ]]; then
        echo "check response \"curl -s ${url}\": should be \"${correct_resp}\" get \"${resp}\""
        slack stops
        exit 1
    fi
}


checkCDN() {
    ip=$1
    checkResponse "http://${ip}:${origin_port}" "Hello, world! This is origin"
    checkResponse '-H "Obj-Type:chunk" -H "Ec-Chunk:4_3_0" http://'${ip}':'${origin_port}'/akamai/ab_24' '********'

    checkResponse "http://${ip}:${ats_port}" "Hello, world! This is origin"
    checkResponse "http://${ip}:${ats_port}/akamai/ab_24" '************************'

    checkResponse "http://${ip}:${fe_port}" "Hello, world I am Frontend!"
    # change this to ab_24 will cause no bucket header error 
    checkResponse '-H Bucket:1 http://'${ip}':'${fe_port}'/akamai/abcd_24' '************************'

    echo -ne "check CDN ${ip}   \t  response pass: \t"
}


checkNodeStatus(){
    info "ts: nReq/nRAM/nHit/nMiss, nReq/nRAM/nHit/nMiss, trafficInterval trafficTotal, err"
    for ip in ${cdn_pub_ips[@]}; do
        ssh -q ${user}@$ip -t 'if [ -f '${OUTPUT_DIR}'/client.stat ]; then echo -ne "\t\t"; tail -n 1 '${OUTPUT_DIR}'/client.stat | tr " " "\t"; fi'
    done 

    echo "check origin http://${origin_ip}:${origin_port}" $(curl -s http://${origin_ip}:${origin_port})

    for ip in ${cdn_pub_ips[@]}; do
        checkCDN ${ip}
        ssh -q ${user}@$ip -tt '''
            echo -e "$(hostname)" 
            echo q | $HOME/CDN/ATSRelease/bin/traffic_top | aha --black --line-fix > /tmp/t.html && python2 $HOME/html2text.py /tmp/t.html |grep Disk;
            # echo q | $HOME/CDN/ATSRelease/bin/traffic_top | aha --line-fix | html2text -width 999|grep Disk;
            # echo q | $HOME/CDN/ATSRelease/bin/traffic_top | aha --line-fix | html2text -width 999|grep "Disk Used"|cut -d " " -f 4
        '''
    done

    for i in `seq 0 $((${#cdn_pub_ips[@]}-1))`; do
        (ssh -q ${user}@${cdn_pub_ips[$i]} -t '''
            if [[ '${i}' -eq 0 ]]; then dstat -D total,nvme0n1,nvme1n1,nvme2n1 -d --disk-util --disk-tps --disk-avgqu --disk-wait --integer --net --top-cpu 1 1| sed "3,3d"; fi
            /usr/bin/dstat -D total,nvme0n1,nvme1n1,nvme2n1 -d --disk-util --disk-tps --disk-avgqu --disk-wait --integer --net --top-cpu 5 2 | sed "1,3d"
        ''') &
    done
    wait
}




        # sudo iotop -o -P -b -n 1 -q|grep DISK | grep -v PID;
        # sudo iostat -m -y -s -z 2 1|grep nvme;


