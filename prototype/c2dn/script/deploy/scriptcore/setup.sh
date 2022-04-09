#!/bin/bash 


import() {
    local curr_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
    source "${curr_dir}/const.sh"
    source "${launch_script_dir}/params.sh"
}

import


function gen_ip_port {
    export ats_pub_ip_port=()
    export ats_priv_ip_port=()
    # this is the one with frontend
    export cdn_pub_ip_port=()
    export cdn_priv_ip_port=()
    export ats_priv_ip_port_str=""
    export cdn_priv_ip_port_str=""

    export node_exp_pub_ip_port=()
    export node_exp_pub_ip_port_str=""

    for ip in ${cdn_pub_ips[@]}; do
        ats_pub_ip_port+=($ip:${ats_port})
        cdn_pub_ip_port+=($ip:${fe_port})
        node_exp_pub_ip_port+=($ip:9100)
    done
    for ip in ${cdn_priv_ips[@]}; do
        ats_priv_ip_port+=($ip:${ats_port})
        cdn_priv_ip_port+=($ip:${fe_port})
    done

    export ats_priv_ip_port_str=$(join_by ${ats_priv_ip_port[@]})
    export cdn_priv_ip_port_str=$(join_by ${cdn_priv_ip_port[@]})
    export cdn_pub_ip_port_str=$(join_by ${cdn_pub_ip_port[@]})
    export node_exp_pub_ip_port_str=$(join_by_comma ${node_exp_pub_ip_port[@]})
}

checkBeforeRun(){
    info "expname $expname, unavail_trace ${unavail_trace}"
    info "use_backuped_disk_cache ${use_backuped_disk_cache}" 
    info "warmup_concurrency ${warmup_concurrency}, warmup_speedup ${warmup_speedup}"
    info "availability_zone $availability_zone"
    info "origin IP ${origin_ip}"
    info "client IP ${client_ip}"
    info "cdn_pub_ips $(echo ${cdn_pub_ips[@]})"
    info "cdn_priv_ips $(echo ${cdn_priv_ips[@]})"

    askContinue 
}


localProcessing() {
    rm ../../../frontend ../../../client ../../../origin ../../../client.status 2>/dev/null || true

    pushd ${top_level_dir}/../ >/dev/null
    [ -f c2dn.tar.gz ] && rm c2dn.tar.gz; 
    tar --exclude='.git' --exclude='.idea' --exclude='__pycache__' --exclude='pkg' --exclude='script/analysis' --exclude='bin' --exclude='build' -zcf c2dn.tar.gz c2dn;
    popd >/dev/null


    rm /tmp/cdnhost_${expname} 2>/dev/null || true
    for ip in ${cdn_pub_ips[@]}; do 
        echo $ip >> /tmp/cdnhost_${expname}
    done 


    info "localProcessing finished" 
}


gatherParams() {
    gen_ip_port

    echo '#!/bin/bash' > ${scriptcore_dir}/init/params.sh
    echo "export system=${system}" >> ${scriptcore_dir}/init/params.sh
    echo "export expname=${expname}" >> ${scriptcore_dir}/init/params.sh

    echo "export mode=${mode}" >> ${scriptcore_dir}/init/params.sh
    echo "export rep_factor=${rep_factor}" >> ${scriptcore_dir}/init/params.sh
    echo "export EC_n=${EC_n}" >> ${scriptcore_dir}/init/params.sh
    echo "export EC_k=${EC_k}" >> ${scriptcore_dir}/init/params.sh

    echo "export ats_priv_ip_port_str=\"${ats_priv_ip_port_str}\"" >> ${scriptcore_dir}/init/params.sh
    echo "export cdn_priv_ip_port_str=\"${cdn_priv_ip_port_str}\"" >> ${scriptcore_dir}/init/params.sh
    echo "export cdn_pub_ip_port_str=\"${cdn_pub_ip_port_str}\"" >> ${scriptcore_dir}/init/params.sh
    echo "export ats_pub_ip_port=(${ats_pub_ip_port[@]})" >> ${scriptcore_dir}/init/params.sh
    echo "export ats_priv_ip_port=(${ats_priv_ip_port[@]})" >> ${scriptcore_dir}/init/params.sh
    echo "export cdn_pub_ip_port=(${cdn_pub_ip_port[@]})" >> ${scriptcore_dir}/init/params.sh
    echo "export cdn_priv_ip_port=(${cdn_priv_ip_port[@]})" >> ${scriptcore_dir}/init/params.sh
    echo "export origin_ip=${origin_ip}" >> ${scriptcore_dir}/init/params.sh


    echo "export warmup_concurrency=${warmup_concurrency}" >> ${scriptcore_dir}/init/params.sh


    echo "export origin_port=${origin_port}" >> ${scriptcore_dir}/init/params.sh
    echo "export ats_port=${ats_port}" >> ${scriptcore_dir}/init/params.sh
    echo "export fe_port=${fe_port}" >> ${scriptcore_dir}/init/params.sh

    echo "export trace_type=${trace_type}" >> ${scriptcore_dir}/init/params.sh
    echo "export server_cache_size=${server_cache_size}" >> ${scriptcore_dir}/init/params.sh
    echo "export ats_ram_size=${ats_ram_size}" >> ${scriptcore_dir}/init/params.sh
    echo "export fe_ram_size=${fe_ram_size}" >> ${scriptcore_dir}/init/params.sh
    echo "export unavail_trace=${unavail_trace}" >> ${scriptcore_dir}/init/params.sh
    echo "export trace_path=${trace_path}" >> ${scriptcore_dir}/init/params.sh

    echo "export eval_start_ts=${eval_start_ts}" >> ${scriptcore_dir}/init/params.sh
    echo "export eval_end_ts=${eval_end_ts}" >> ${scriptcore_dir}/init/params.sh
    echo "export use_remote_origin=${use_remote_origin}" >> ${scriptcore_dir}/init/params.sh
    # echo "export warmup_uniq_ram_trace_src=${warmup_uniq_ram_trace_src}" >> ${scriptcore_dir}/init/params.sh
    # echo "export warmup_uniq_disk_trace_src=${warmup_uniq_disk_trace_src}" >> ${scriptcore_dir}/init/params.sh

    echo "export testbed=${testbed}" >> ${scriptcore_dir}/init/params.sh


    echo "export DEBUG_MODE=${DEBUG_MODE:-0}" >> ${scriptcore_dir}/init/params.sh
} 


startPrometheus() {
    pkill -f prometheus 2>/dev/null || true; 
    gen_ip_port

    sleep 2;
    pushd /tmp/prometheus-2.24.0.linux-amd64 >/dev/null || true;

    if [ ! -f prometheus ]; then 
        cd /tmp/;
        wget https://github.com/prometheus/prometheus/releases/download/v2.24.0/prometheus-2.24.0.linux-amd64.tar.gz
        tar xvf prometheus-2.24.0.linux-amd64.tar.gz
        cd /tmp/prometheus-2.24.0.linux-amd64;
    fi

    echo '''
global:
  scrape_interval:   20s 

scrape_configs:
  - job_name: CDN
    scrape_interval: 5s
    static_configs:
      - targets: ['${node_exp_pub_ip_port_str}']

  - job_name: origin
    scrape_interval: 10s
    static_configs:
      - targets: ['${origin_ip}':9100]

  - job_name: client
    scrape_interval: 10s
    static_configs:
      - targets: ['${client_ip}':9100]

''' > /tmp/prometheus.yml     
    screen -L -Logfile /tmp/prometheus.screen -S prom -dm ./prometheus --storage.tsdb.retention.time 30d --config.file=/tmp/prometheus.yml 

    popd > /dev/null; 
    info "prometheus started"
}

prepareCDN() {
    init_cdn=0
    if [ ${#cdn_pub_ips[@]} -eq 0 ]; then
        info "create CDN instances:     python3 ${scriptcore_dir}/awsVMs.py CDN ${expname} ${placement_group} ${availability_zone}"
        python3 ${scriptcore_dir}/awsVMs.py CDN ${expname} ${placement_group} ${availability_zone}
        IFS=$'\n'
        for line in `cat /tmp/cdnPub`; do cdn_pub_ips+=($line); done
        for line in `cat /tmp/cdnPrv`; do cdn_priv_ips+=($line); done
        init_cdn=1
    fi

    gatherParams        # this is needed because CDN ip are not available till now



    if [ $init_cdn = 1 ] || [ $FORCE_INIT = 1 ]; then 
        info "initialize CDN"
        for i in `seq 0 $((${#cdn_pub_ips[@]}-1))`; do
            sleep 0.2
            (   
                ssh -q ${user}@${cdn_pub_ips[${i}]} "rm -r /tmp/c2dn * 2>/dev/null || true"
                rsync -r $HOME/.aws ${user}@${cdn_pub_ips[${i}]}: ;
                rsync -r $HOME/.ssh ${user}@${cdn_pub_ips[${i}]}: ;
                rsync -r ${scriptcore_dir}/init ${user}@${cdn_pub_ips[${i}]}:/tmp/;
                rsync -r --exclude="*.git*" ${top_level_dir} ${user}@${cdn_pub_ips[${i}]}:${remote_exp_dir};
                ssh -q ${user}@${cdn_pub_ips[${i}]} -tt '''
                    cd /tmp/init/; echo export nodeIdx='${i}' >> params.sh; echo export clientIdx='${i}' >> params.sh; 
                    chmod +x *.sh; bash ./init_cdn.sh -e '${expname}' --stop --init --build --start;
                ''' || exit
            ) &
        done
        wait

        info "initialize CDN servers done"

    else 
        info "re-setup CDN"
        for i in `seq 0 $((${#cdn_pub_ips[@]}-1))`; do
            sleep 0.2
            (
                ssh -q ${user}@${cdn_pub_ips[${i}]} "rm -r /tmp/c2dn/* 2>/dev/null || true"
                rsync -r ${scriptcore_dir}/init ${user}@${cdn_pub_ips[${i}]}:/tmp/;
                rsync -r --exclude="*.git*" ${top_level_dir} ${user}@${cdn_pub_ips[${i}]}:${remote_exp_dir};
                ssh -q ${user}@${cdn_pub_ips[${i}]} '''
                    cd /tmp/init/; echo export nodeIdx='${i}' >> params.sh; echo export clientIdx='${i}' >> params.sh; 
                    chmod +x *.sh; bash ./init_cdn.sh -e '${expname}' --stop --build --start;
                ''' || exit
            ) &
        done
        wait

        info "re-setup CDN servers done"
    fi 


    startPrometheus
}


prepareRemoteOrigin(){
    gatherParams

    init_origin=0
    if [[ "$origin_ip" == "0.0.0.0" ]] || [[ "$origin_ip" == "" ]]; then
        info "create origin instance"
        python3 ${scriptcore_dir}/awsVMs.py origin ${expname}
        export origin_ip=`cat /tmp/origin`;
        init_origin=1
    fi 


    if [ $init_origin = 1 ] || [ $FORCE_INIT = 1 ]; then 
        ssh -q ${user}@${origin_ip} "rm -r /tmp/c2dn/* 2>/dev/null || true"
        info "initialize remote origin"
        rsync -r $HOME/.aws ${user}@${origin_ip}: ;
        rsync -r ${scriptcore_dir}/init ${user}@${origin_ip}:/tmp/;
        rsync -r --exclude="*.git*" ${top_level_dir} ${user}@${origin_ip}:${remote_exp_dir};
        ssh -q ${user}@${origin_ip} -tt ''' 
            cd /tmp/init/; chmod +x *.sh; bash ./init_origin.sh -e '${expname}' --stop --init --build --start;
        ''' || exit
    fi


    # if [[ "${system}" == "C2DN" ]]; then
        info "set origin no cache"
        curl -s "http://${origin_ip}:${origin_port}/setNoCache" && echo 
    # fi 
    info "remote origin ${origin_ip} ready"
}


prepareRemoteClient() {
    init_client=0
    if [[ "$client_ip" == @(""|"127.0.0.1") ]]; then
        info "create remote client instance "
        python3 ${scriptcore_dir}/awsVMs.py client ${expname}
        export client_ip=$(cat /tmp/client);
        init_client=1
    fi

    gatherParams        # this is needed because client ip is not available till now

    # set to 1 because we want to update client every time 
    init_client=1

    if [ $init_client = 1 ] || [ $FORCE_INIT = 1 ]; then 
        ssh -q ${user}@${client_ip} "rm -r /tmp/c2dn/* * 2>/dev/null || true"
        info "initialize remote client"
        rsync -r $HOME/.aws ${user}@${client_ip}: ;
        rsync -r ${scriptcore_dir}/init ${user}@${client_ip}:/tmp/;
        rsync -r --exclude=*.git* ${top_level_dir} ${user}@${client_ip}:${remote_exp_dir}; 
        ssh -q ${user}@${client_ip} -tt '''
            cd /tmp/init/; echo export nodeIdx=-1 >> params.sh; echo export clientIdx=-1 >> params.sh; 
            chmod +x *.sh; bash ./init_client.sh -e '${expname}' --stop --init --build; 
        ''' || exit
    fi
}




