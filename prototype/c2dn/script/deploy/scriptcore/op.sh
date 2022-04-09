#!/bin/bash 


import() {
    local curr_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
    source "${curr_dir}/const.sh"
    source "${curr_dir}/utils.sh"
    source "${curr_dir}/monitor.sh"
    source "${launch_script_dir}/params.sh"
}

import


getCDNips(){
    python3 ${DIR}/awsVMs.py getCDNips $expname
}


_warmup(){
    # warmup traceLoc originType
    info "start cache warmup procedure with start_ts ${1:-} end_ts ${2:-} unique_obj ${3:-} use_remote_origin ${use_remote_origin} ignore_remote_req 0 speedup ${warmup_speedup} concurrency ${warmup_concurrency}"
    start=$1
    end=$2
    unique_obj=$3
    ignore_remote_req=0
    random_route=0

    # [[ ! "$1" == @("non_uniq"|"uniq_disk"|"uniq_ram") ]] && echo unsupported trace type $1 && exit 1;
    # [[ ! "$5" == @("remote_origin"|"local_origin") ]] && echo unsupported origin type $5 && exit 1;

    # for i in `seq 0 $((${#cdn_pub_ips[@]}-1))`; do
    #     ssh -q ${user}@${cdn_pub_ips[${i}]} -t "cd /tmp/init/; bash cdn_op.sh --replayTrace ${start} ${end} ${unique_obj} ${use_remote_origin} ${ignore_remote_req} ${warmup_speedup} ${warmup_concurrency}" || exit
    # done


    parallel-ssh -i -t 0 --user ${user} -p ${#cdn_pub_ips[@]} -h /tmp/cdnhost_${expname} "
        cd /tmp/init/; bash cdn_op.sh --replayTrace ${start} ${end} ${unique_obj} ${use_remote_origin} ${ignore_remote_req} ${warmup_speedup} ${warmup_concurrency} ${random_route}
    " 
}


warmupCluster(){
    # warmupCluster originType
    # if [[ "$1" != remoteOrigin ]] && [[ "$1" != localOrigin ]] && [[ "$1" != supportBucket ]]; then
    #     echo unsupported origin type $1;
    #     exit;
    # fi
    info "start cache warmup"
    unique_obj=0
    curr_stage="warmup"
    if [ "${use_backuped_disk_cache}" == 1 ]; then
        curr_stage="backuped_disk_cache_ram_warmup"
        _warmup ${warmup_ram_start_ts} ${warmup_ram_end_ts} ${unique_obj} 
    elif [ "${uniq_obj_warmup}" == 1 ]; then
        curr_stage="disk_warmup"
        _warmup ${warmup_disk_start_ts} ${warmup_disk_end_ts} 1 
        waitForWarmupFinish
        askContinue "Disk cache warmup finished, next is ram cache warmup"
        curr_stage="ram_warmup"
        _warmup ${warmup_ram_start_ts} ${warmup_ram_end_ts} 0 
    else
        _warmup ${warmup_disk_start_ts} ${warmup_ram_end_ts} ${unique_obj} 
    fi
    waitForWarmupFinish
    askContinue "warmup finished, next is to start client"
}


waitForWarmupFinish() {
    waitForLocalClientToFinish
}

waitForLocalClientToFinish(){
    finishedIPs=()
    while [ ${#finishedIPs[@]} -lt ${#cdn_pub_ips[@]} ]; do
        finishedIPs=()
        sleep 20
        for ip in ${cdn_pub_ips[*]}; do
            local status=$(ssh -q ${user}@$ip "[ -f ${LOCAL_WORKING_DIR}/client.status ] && echo 1 || echo 0")
            [ $status = 1 ] && finishedIPs+=($ip)
        done
        clear
        info "${expname} curr_stage ${curr_stage:-} warmup_end ${warmup_ram_end_ts} eval_end ${eval_end_ts}"
        info "${#finishedIPs[*]} clients have finished ${finishedIPs[@]}"
        checkNodeStatus
    done
    info "all local clients has finished"
}









prepareForEval() {
    info "reset origin and frontend stat"
    parallel-ssh -i -t 0 --user ${user} -h /tmp/cdnhost_${expname} '''
        # reset origin stat
        curl -so /dev/null 127.0.0.1:'${origin_port}'/reset/
        # reset frontend stat
        curl -so /dev/null 127.0.0.1:'${fe_port}'/reset/
        curl -so /dev/null 127.0.0.1:'${fe_port}'/startUnavailReplay/
    ''' 
}




runLocalClient() {
    speedup=$1

    unique_obj=0
    ignore_remote_req=1
    concurrency=32
    random_route=1
    [[ "${mode}" == "noRep" ]] && random_route=0 || true

    info "start local stress clients with speedup ${speedup} unique_obj ${unique_obj} concurrency ${concurrency} random_route ${random_route}" 
    parallel-ssh -i -t 0 --user ${user} -p ${#cdn_pub_ips[@]} -h /tmp/cdnhost_${expname} "
        cd /tmp/init/; 
        echo 'eval start $(date +%s) > /tmp/c2dn/ts'
        bash cdn_op.sh --startMonitoring
        bash cdn_op.sh --replayTrace ${eval_start_ts} ${eval_end_ts} ${unique_obj} ${use_remote_origin} ${ignore_remote_req} ${speedup} ${concurrency} ${random_route}
    "
}




runRemoteClient() {
    speedup=$1
    random_route=1
    [[ "${mode}" == "noRep" ]] && random_route=0 || true

    info "start remote client ${client_ip} with speedup ${speedup}" 
    ssh -q ${user}@${client_ip} "cd /tmp/init/; bash ./init_client.sh -e ${expname} --start ${speedup} ${random_route}" 

    curr_stage="eval"
}


waitForRemoteClientToFinish() {
    info "wait for remote client ${client_ip} to finish"

    echo -n wait 
    finished=0
    while [ $finished = 0 ]; do 
        local status=$(ssh -q ${user}@${client_ip} "[ -f ${LOCAL_WORKING_DIR}/client.status ] && echo 1 || echo 0")
        [ "$status" = 1 ] && finished=1
        sleep 2
        echo -n '.'
    done 
    echo
}

waitForEval(){
    waitForLocalClientToFinish
    waitForRemoteClientToFinish

    info "all eval clients finished"
}

stopExp() {
    parallel-ssh -i -t 0 --user ${user} -p ${#cdn_pub_ips[@]} -h /tmp/cdnhost_${expname} "
        echo -ne stop >/tmp/c2dn/monitorCmd;
    
        curl -so /tmp/c2dn/metricFE 127.0.0.1:2022/metrics
        curl -so /tmp/c2dn/metrics_origin 127.0.0.1:2020/metrics
        ./CDN/ATSRelease/bin/traffic_logstats > /tmp/c2dn/ats_stat
    "

    ssh ${user}@${origin_ip} '''
        echo -ne stop >/tmp/c2dn/monitorCmd;
        curl -so /tmp/c2dn/metrics_origin 127.0.0.1:2020/metrics
    '''

    sleep 20 
}

collectResult() {
    info "upload results" 
    aws s3 cp ${top_level_dir}/../c2dn.tar.gz s3://juncheng-data/C2DN/prototype/$(date +%Y-%m-%d)/${expname}/

    # copy array 
    allips=("${cdn_pub_ips[@]}")

    # collect CDN
    for i in `seq 0 $((${#cdn_pub_ips[@]}-1))`; do
        (ssh -q ${user}@${cdn_pub_ips[${i}]} '''
            zip -rq $HOME/c2dn_cdn_'${i}'.zip /tmp/c2dn/
            aws s3 cp $HOME/c2dn_cdn_'${i}'.zip s3://juncheng-data/C2DN/prototype/$(date +%Y-%m-%d)/'${expname}'/
        ''') &
    done 
    wait 

    ssh -q ${user}@${origin_ip} '''
            zip -rq $HOME/c2dn_origin.zip /tmp/c2dn/
            aws s3 cp $HOME/c2dn_origin.zip s3://juncheng-data/C2DN/prototype/$(date +%Y-%m-%d)/'${expname}'/
    '''

    ssh -q ${user}@${client_ip} '''
            zip -rq $HOME/c2dn_client.zip /tmp/c2dn/
            aws s3 cp $HOME/c2dn_client.zip s3://juncheng-data/C2DN/prototype/$(date +%Y-%m-%d)/'${expname}'/
    '''
    curl -X POST -H 'Content-type: application/json' --data '{"text":"'${expname}' finish"}' ${slackhook}
}


backupDiskCache() {
    info "backup disk cache ${expname}" 

    for i in `seq 0 $((${#cdn_pub_ips[@]}-1))`; do
        ip=${cdn_pub_ips[i]}
        (ssh -q ${user}@$ip -tt '''
            $HOME/CDN/ATSRelease/bin/trafficserver stop; sleep 20;

            # aws s3 cp /disk0/t0/cache.db s3://juncheng-data/C2DNDiskCache/'${expname}'/cache.db.'$i'.disk0.t0 &
            # aws s3 cp /disk1/t0/cache.db s3://juncheng-data/C2DNDiskCache/'${expname}'/cache.db.'$i'.disk1.t0 &

            mkdir /disk0/backup2 /disk1/backup2 2>/dev/null || true;
            cp -r /disk0/t0/cache.db /disk0/backup2/'${expname}'.db &
            cp -r /disk1/t0/cache.db /disk1/backup2/'${expname}'.db &
            wait

            $HOME/CDN/ATSRelease/bin/trafficserver start; sleep 20;
        ''') &
    done 
    wait 
}


restoreDiskCache() {
    info "restore disk cache ${expname}" 

    for i in `seq 0 $((${#cdn_pub_ips[@]}-1))`; do
        ip="${cdn_pub_ips[i]}"
        (echo -n -e $ip '\t' ;
        ssh -q ${user}@$ip -tt '''echo $(hostname)
            $HOME/CDN/ATSRelease/bin/trafficserver stop; sleep 20;
            mkdir /disk0/t0 /disk1/t0 2>/dev/null || true;

            cp /disk0/backup2/'${expname}'.db /disk0/t0/cache.db &
            cp /disk1/backup2/'${expname}'.db /disk1/t0/cache.db &
            wait

            $HOME/CDN/ATSRelease/bin/trafficserver start; sleep 20;
        ''') &
    done 
    wait 
}







