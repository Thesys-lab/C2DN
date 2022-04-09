#!/bin/bash 


DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source "${DIR}/init_utils.sh"
source "${DIR}/init_func.sh"
source "${DIR}/params.sh"


replayTrace() {
    # warmup_trace=${LOCAL_TRACE_DIR}/akamai.bin
    [[ "${nodeIdx}" == "" ]] && error "replay trace nodeIdx empty, please check cdn local params.sh"

    pushd ${C2DN_DIR} >/dev/null
    rm ${C2DN_DIR}/client.status 2>/dev/null || true
    debug "./client -mode=replayOpenloop -trace=${LOCAL_TRACE_DIR}/akamai.bin -replayStartTs=${start} -replayEndTs=${end} -ignoreRemoteReq=${ignore_remote_req} -uniqueObj=${unique_obj} -remoteOrigin=${remote_origin} -randomRoute=${random_route} -clientID=${nodeIdx} -concurrency=${concurrency} -replaySpeedup=${speedup} -nServers=10 ${cdn_pub_ip_port_str}"
    screen -S replay -L -Logfile /tmp/c2dn/screen/replay_${start}_${end} -dm ./client -mode=replayOpenloop -trace=${LOCAL_TRACE_DIR}/akamai.bin -replayStartTs=${start} -replayEndTs=${end} -ignoreRemoteReq=${ignore_remote_req} -uniqueObj=${unique_obj} -remoteOrigin=${remote_origin} -randomRoute=${random_route} -clientID=${nodeIdx} -concurrency=${concurrency} -replaySpeedup=${speedup} -nServers=10 ${cdn_pub_ip_port_str}
    popd >/dev/null

    echo $(date) ${FUNCNAME[0]} ${1:-} done >> ${status_log}
} 



checkStatus() {
    pub_ip=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)

    # echo -n -e "$(hostname), \t ${pub_ip}: \t"
    # [ -f $HOME/CDN/c2dn/client.stat ] && tail -n 1 $HOME/CDN/c2dn//client.stat | tr "\n" "\t"; 
    # echo -n "origin: ";
    # curl 127.0.0.1:'${origin_port}'/akamai/ab_25; echo -n ", "
    # echo -n "ats: ";
    # curl 127.0.0.1:'${ats_port}'/akamai/ab_25; echo -n ", "
    # echo -n "frontend: "; curl 127.0.0.1:'${fe_port}'/akamai/ab_25; echo

    # # dstat -d --disk-util --disk-tps --disk-avgqu --disk-wait --net --top-cpu 1 1| sed "3,3d"
    # # /usr/bin/dstat -d --disk-util --disk-tps --disk-avgqu --disk-wait --net --top-cpu 1 2| sed "1,3d";

    # # echo q | $HOME/CDN/ATSRelease/bin/traffic_top | aha --line-fix | html2text -width 999|grep Disk;
    # echo q | $HOME/CDN/ATSRelease/bin/traffic_top | aha --black --line-fix > /tmp/t.html && python2 $HOME/html2text.py /tmp/t.html |grep Disk;

    # sudo iotop -o -P -b -n 1 -q|grep DISK | grep -v PID;
    # sudo iostat -m -y -s -z 2 1|grep nvme;
    # echo q | $HOME/CDN/ATSRelease/bin/traffic_top | aha --line-fix | html2text -width 999|grep "Disk Used"|cut -d " " -f 4


}

replay=0
start=0
end=0
unique_obj=0
remote_origin=0
# warmup should not ignore request for remote origin, evaluation should ignore requests for remote origin
ignore_remote_req=1
speedup=1
concurrency=1
random_route=1


while [ "${1:-}" != "" ]; do
    case "$1" in
        "--replayTrace")
            replay=1
            start=$2
            end=$3
            unique_obj=$4
            remote_origin=$5
            ignore_remote_req=$6
            speedup=$7
            concurrency=$8
            random_route=$9
            shift
            shift
            shift
            shift
            shift
            shift
            shift
            shift
            ;;
        "-n" | "--nodeIdx")
            nodeIdx=$2
            shift
            ;;
        "--startMonitoring")
            startMonitoring
            ;;
    esac
    shift
done 



[ ${replay} != 0 ] && replayTrace 











