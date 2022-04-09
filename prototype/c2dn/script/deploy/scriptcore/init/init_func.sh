#!/bin/bash
source params.sh
export DEBIAN_FRONTEND=noninteractive

PATH=$PATH:/usr/local/go/bin;
# GOPATH=$HOME/CDN/c2dn/;
# GOBIN=$HOME/CDN/c2dn/;
EXP_DIR=$HOME/CDN/
ATS_DIR=${EXP_DIR}/ATSRelease/
C2DN_DIR=${EXP_DIR}/c2dn/
LOCAL_TRACE_DIR=$HOME/data/
status_log=$HOME/status


setHostname() {
    hostname=${1:-myhost}
    sudo hostnamectl set-hostname ${hostname}
    # echo ${hostname} | sudo tee /etc/hostname >/dev/null;
    # sudo sed -i "s/localhost/${hostname}/g" /etc/hosts || true;
}

initDir() {
    mkdir -p /tmp/c2dn/log /tmp/c2dn/screen /tmp/c2dn/output 2>/dev/null || true;
    mkdir -p $HOME/software/source 2>/dev/null || true;
    mkdir -p ${C2DN_DIR} 2>/dev/null || true;
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}

wipeDir() {
    rm -r /tmp/c2dn  2>/dev/null || true;
    rm -r ${ATS_DIR} 2>/dev/null || true; 

    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}


initSettings() {
    grep ${FUNCNAME[0]} ${status_log} > /dev/null && return 

    echo "$(whoami) ALL=(ALL:ALL) NOPASSWD: ALL" | sudo tee -a /etc/sudoers >/dev/null
    sudo usermod -a -G disk $(whoami)

    # mkdir -p $HOME/.aws/ 2>/dev/null || true; mkdir -p $HOME/.ssh/ 2>/dev/null || true;
    # echo '[default]' > $HOME/.aws/config; echo 'region = us-east-1' >> $HOME/.aws/config; echo  >> $HOME/.aws/config;

    echo "export PATH=$PATH:/usr/local/go/bin/:${ATS_DIR}/bin/:${C2DN_DIR}/;" > $HOME/.bashrc
    # echo "export PATH=$PATH:/usr/local/go/bin:${ATS_DIR}/bin/:$HOME/CDN/c2dn/; export GOPATH=$HOME/CDN/c2dn; export GOBIN=$HOME/CDN/c2dn;" > $HOME/.bash_profile
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}


installPackages() {
    grep ${FUNCNAME[0]} ${status_log} > /dev/null && return

    sudo add-apt-repository -y ppa:ubuntu-toolchain-r/test >/dev/null
    sudo apt-get -qq -o=Dpkg::Use-Pty=0 update >/dev/null
    sudo apt-get -qq -o=Dpkg::Use-Pty=0 install -y htop bc ntp ntpdate git awscli bmon iotop zip sysstat iftop dstat ifstat atop ioping >/dev/null
    sudo apt-get -qq -o=Dpkg::Use-Pty=0 install -y python3 python3-dev python3-matplotlib python3-pip >/dev/null
    sudo apt-get -qq -o=Dpkg::Use-Pty=0 install -y gcc g++ make cmake autogen autoconf automake libtool build-essential pkg-config >/dev/null
    sudo apt-get -qq -o=Dpkg::Use-Pty=0 install -y  aha html2text mosh e2fsprogs libunwind-dev >/dev/null
    sudo apt-get -qq -o=Dpkg::Use-Pty=0 install -y libmodule-install-perl libssl-dev libpcre3-dev libcap-dev libhwloc-dev libncurses5-dev libcurl4-openssl-dev flex tcl-dev >/dev/null
    sudo apt-get -qq -o=Dpkg::Use-Pty=0 install -y tuned >/dev/null
    sudo -H pip3 install psutil >/dev/null



    # sudo apt-get -qq install -y libboost-all-dev 
    # sudo apt-get -qq install -y linux-tools-common linux-tools-generic linux-tools-5.4.0-58-generic

    # sudo apt-get -qq -o=Dpkg::Use-Pty=0 install -y yasm nasm
    # sudo apt-get -qq -o=Dpkg::Use-Pty=0 install -y libedit-dev liberasurecode-dev libjerasure-dev libjerasure2
    # sudo apt-get -qq -o=Dpkg::Use-Pty=0 install -y 

    # sudo pip3 install -qU awscli numpy matplotlib pyeclib flask gunicorn greenlet eventlet gunicorn[eventlet] gevent gunicorn[gevent]
    # sudo apt-get install -yqq gunicorn >/dev/null

    # sudo rm /usr/bin/python 2>/dev/null || true; sudo ln -s /usr/bin/python3 /usr/bin/python;
    # sudo rm /usr/bin/gcc /usr/bin/g++;
    # sudo ln -s /usr/bin/gcc-8 /usr/bin/gcc;
    # sudo ln -s /usr/bin/g++-8 /usr/bin/g++;

    rm $HOME/html2text.py 2>/dev/null || true; 
    cd $HOME; wget -q https://raw.githubusercontent.com/aaronsw/html2text/master/html2text.py >/dev/null;
    chmod +x $HOME/html2text.py; sed -i "s/text = text.encode('utf-8')/text = text.encode('utf-8').strip()/g" $HOME/html2text.py;

    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}

synctime() {
    grep ${FUNCNAME[0]} ${status_log} > /dev/null && return

    sudo timedatectl set-timezone America/New_York;
    sudo service ntp stop >/dev/null
    sudo ntpdate pool.ntp.org >/dev/null
    sudo service ntp start >/dev/null
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}

prepareDisk() {
    grep ${FUNCNAME[0]} ${status_log} > /dev/null && return

    rm -r disk1 disk2 disk3 disk4 2>/dev/null || true
    mkdir disk1 disk2 disk3 disk4 2>/dev/null || true
    echo -e "o\nn\np\n1\n\n\nw" | fdisk /dev/nvme1n1
    echo -e "y\n\n" | mkfs.ext4 /dev/nvme1n1
    mount /dev/nvme1n1 disk1
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}

prepareDiskAWS() {
    nDisk=2
    nFolder=1
    cache_size_GiB=$(echo ${server_cache_size}/1.024/1.024/1.024/${nDisk}/${nFolder}|bc)

    echo why do I have to enable multiple folders
    for j in `seq 0 $((nDisk-1))`; do
        if [[ -z "$(lsblk |grep disk${j})" ]]; then
            sudo mkdir /disk${j} 2>/dev/null || true;
            echo -e 'y\n\n' | sudo mkfs.ext4 /dev/nvme${j}n1 >/dev/null;
            sleep 2;
            sudo tune2fs -f -O ^has_journal /dev/nvme${j}n1 >/dev/null;
            sudo mount -t ext4 -O noatime,data=writeback,barrier=0 /dev/nvme${j}n1 /disk${j}
            sudo chown -R $(whoami) /disk${j}
        fi
    done


    rm ${ATS_DIR}/etc/trafficserver/storage.config 2>/dev/null || true;

    for m in `seq 0 $((nDisk-1))`; do
        for j in `seq 0 $((nFolder-1))`; do
            echo /disk${m}/t${j} ${cache_size_GiB}G >> ${ATS_DIR}/etc/trafficserver/storage.config; 
            mkdir /disk${m}/t${j} 2>/dev/null || true;
            sudo chown -R $(whoami) /disk${j}
        done
    done

    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
} 


performanceTuning() {
    # echo -e "*\t-\tnofile\t65536" | sudo tee /etc/security/limits.conf >/dev/null 
    echo 5 | sudo tee /proc/sys/net/ipv4/tcp_fin_timeout >/dev/null
    # echo 1 | sudo tee /proc/sys/net/ipv4/tcp_tw_recycle >/dev/null
    echo 1 | sudo tee /proc/sys/net/ipv4/tcp_tw_reuse >/dev/null

    # echo "net.core.rmem_max = 33554432" | sudo tee -a /etc/sysctl.conf > /dev/null 
    # echo "net.core.wmem_max = 33554432" | sudo tee -a /etc/sysctl.conf > /dev/null 

    # echo "net.ipv4.tcp_rmem = 4096 87380 33554432" | sudo tee -a /etc/sysctl.conf > /dev/null 
    # echo "net.ipv4.tcp_wmem = 4096 65536 33554432" | sudo tee -a /etc/sysctl.conf > /dev/null 

    # sudo cpupower frequency-set -g performance

    sudo tuned-adm profile throughput-performance
    # sudo tuned-adm profile network-latency
    # sudo tuned-adm profile virtual-guest
    # sudo tuned-adm profile latency-performance

    # ifconfig eth0 txqueuelen 2000

    info "performanceTuning turned off"

    sudo sysctl -p > /dev/null 
}

installATS(){
    grep ${FUNCNAME[0]} ${status_log} > /dev/null && return

    pushd $HOME/software/source/ >/dev/null;
    wget -q https://apache.claz.org/trafficserver/trafficserver-8.1.1.tar.bz2;
    tar xf trafficserver-8.1.1.tar.bz2;
    cd trafficserver-8.1.1;


    # change aggregation write buffer size, this will cause ATS to crash on large object hits
    # for i in `seq 0 1`; do curl -svo /dev/null 127.0.0.1:8080/akamai/7742_953610202 & done
    # sed -i 's/AGG_SIZE (4 \* 1024 \* 1024)/AGG_SIZE (256 * 1024 * 1024)/g' iocore/cache/P_CacheVol.h
    ./configure --prefix=${ATS_DIR}/ >/dev/null
    make -j >/dev/null
    make install >/dev/null
    cp -r ${ATS_DIR}/etc/trafficserver ${ATS_DIR}/etc/trafficserver.bak

    popd > /dev/null;
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}


installISAL(){
    grep ${FUNCNAME[0]} ${status_log} > /dev/null && return

    pushd $HOME/software/source/ >/dev/null;
    wget -q https://github.com/01org/isa-l/archive/v2.25.0.tar.gz -O isa-v2.25.0.tar.gz;
    tar xf isa-v2.25.0.tar.gz;
    cd isa-l-2.25.0;
    ./autogen.sh >/dev/null
    ./configure --prefix=/usr --libdir=/usr/lib  >/dev/null
    make -j >/dev/null
    sudo make install >/dev/null

    # setup disk permission issue 
    echo SUBSYSTEM=="block", KERNEL=="sd[a-z][0-9]", GROUP:=${group} | sudo tee /etc/udev/rules.d/51-cache-disk.rules >/dev/null;
    echo SUBSYSTEM=="block", KERNEL=="nvme[0-9]n[0-9]", GROUP:=${group} | sudo tee -a /etc/udev/rules.d/51-cache-disk.rules >/dev/null;
    sudo usermod -a -G sudo $(whoami)
    sudo usermod -a -G disk $(whoami)

    popd > /dev/null;
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}


installGO() {
    grep ${FUNCNAME[0]} ${status_log} > /dev/null && return
    pushd $HOME/software/source/ >/dev/null;

    if [[ "$(dpkg --print-architecture)" == amd64 ]]; then
        wget -q https://dl.google.com/go/go1.15.6.linux-amd64.tar.gz -O go.tar.gz;
    elif [[ "$(dpkg --print-architecture)" == arm64 ]]; then
        wget -q https://dl.google.com/go/go1.15.6.linux-arm64.tar.gz -O go.tar.gz;
    fi
    sudo tar -C /usr/local -xzf go.tar.gz

    echo -e "export PATH=$PATH:/usr/local/go/bin\n" >> $HOME/.bashrc

    popd > /dev/null;
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}

installNodeExporter() {
    grep ${FUNCNAME[0]} ${status_log} > /dev/null && return
    pushd $HOME/software/source/ >/dev/null;

    wget https://github.com/prometheus/node_exporter/releases/download/v1.0.1/node_exporter-1.0.1.linux-amd64.tar.gz
    tar xvfz node_exporter-*.*-amd64.tar.gz
    cd node_exporter-*.*-amd64
    mv node_exporter ${EXP_DIR};

    popd > /dev/null;
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}


setupATS() {
    # aws specific 
    pub_ip=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)
    priv_ip=$(curl -s http://169.254.169.254/latest/meta-data/local-ipv4)
    group=$(groups|cut -d " " -f 1)
    local conf_path=${ATS_DIR}/etc/trafficserver/

    rm ${conf_path}/cache.config ${conf_path}/remap.config 2>/dev/null || true

    # setup dram cache size 
    sed -i "s/proxy.config.cache.ram_cache.size INT -1/proxy.config.cache.ram_cache.size INT ${ats_ram_size}/g" ${conf_path}/records.config || true

    grep "sock_option_flag_in" ${conf_path}/records.config >/dev/null || echo "CONFIG proxy.config.net.sock_option_flag_in INT 30000" >> ${conf_path}/records.config
    grep "sock_option_flag_out" ${conf_path}/records.config >/dev/null || echo "CONFIG proxy.config.net.sock_option_flag_out INT 30000" >> ${conf_path}/records.config
    grep "origin_min_keep_alive_connections" ${conf_path}/records.config >/dev/null || echo "CONFIG proxy.config.http.origin_min_keep_alive_connections INT 32" >> ${conf_path}/records.config
    # grep "chunking_enabled" ${conf_path}/records.config >/dev/null || echo "CONFIG proxy.config.http.chunking_enabled INT 0" >> ${conf_path}/records.config
    grep "ip_allow.filename" ${conf_path}/records.config >/dev/null || echo "CONFIG proxy.config.cache.ip_allow.filename STRING ip_allow.config" >> ${conf_path}/records.config
    grep "stats_over_http.so" ${conf_path}/plugin.config >/dev/null || echo "stats_over_http.so" >> ${conf_path}/plugin.config;

    # setup mapping 
    echo "dest_domain=127.0.0.1 port=${origin_port} revalidate=240h" >> ${conf_path}/cache.config;
    echo "dest_domain=${origin_ip} port=${origin_port} revalidate=240h" >> ${conf_path}/cache.config;
    echo "map http://127.0.0.1:${ats_port}/remote/ http://${origin_ip}:${origin_port}/" >> ${conf_path}/remap.config;
    echo "map http://${pub_ip}:${ats_port}/remote/ http://${origin_ip}:${origin_port}/" >> ${conf_path}/remap.config;
    echo "map http://${priv_ip}:${ats_port}/remote/ http://${origin_ip}:${origin_port}/" >> ${conf_path}/remap.config;
    echo "map http://127.0.0.1:${ats_port}/ http://127.0.0.1:${origin_port}/" >> ${conf_path}/remap.config;
    echo "map http://${pub_ip}:${ats_port}/ http://127.0.0.1:${origin_port}/" >> ${conf_path}/remap.config;
    echo "map http://${priv_ip}:${ats_port}/ http://127.0.0.1:${origin_port}/" >> ${conf_path}/remap.config;

    # setup misc
    sed -i "s/proxy.config.http.insert_response_via_str INT 0/proxy.config.http.insert_response_via_str INT 2/g" ${conf_path}/records.config || true
    sed -i "s/proxy.config.http.cache.required_headers INT 2/proxy.config.http.cache.required_headers INT 0/g" ${conf_path}/records.config || true
    # disabled, otherwise push to 127.0.0.1 and localIP will be different
    # sed -i "s/proxy.config.url_remap.pristine_host_hdr INT 0/proxy.config.url_remap.pristine_host_hdr INT 1/g" ${conf_path}/records.config
    sed -i "s/proxy.config.http.keep_alive_no_activity_timeout_out INT 120/proxy.config.http.keep_alive_no_activity_timeout_out INT 30000/g" ${conf_path}/records.config || true
    sed -i "s/proxy.config.http.keep_alive_no_activity_timeout_in INT 120/proxy.config.http.keep_alive_no_activity_timeout_in INT 30000/g" ${conf_path}/records.config || true
    # sed -i "s/proxy.config.cache.ram_cache_cutoff INT 4194304/proxy.config.cache.ram_cache_cutoff INT 131072/g" ${conf_path}/records.config
    sed -i "s/proxy.config.http.push_method_enabled INT 0/proxy.config.http.push_method_enabled INT 1/g" ${conf_path}/records.config || true
    sed -i "s/proxy.config.http.slow.log.threshold INT 0/proxy.config.http.slow.log.threshold INT 8000/g" ${conf_path}/records.config || true
    sed -i "s/proxy.config.cache.limits.http.max_alts INT 5/proxy.config.cache.limits.http.max_alts INT 0/g" ${conf_path}/records.config || true
    sed -i "s/proxy.config.http.cache.when_to_revalidate INT 0/proxy.config.http.cache.when_to_revalidate INT 3/g" ${conf_path}/records.config || true

    echo "src_ip=0.0.0.0-255.255.255.255                    action=ip_allow  method=ALL" > ${conf_path}/ip_allow.config


    udevadm trigger --subsystem-match=block
    echo "$(date) disk permission updated" >> ${status_log}

    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}


setupATS0() {
    # setup disk


    cp -n ${ATS_DIR}/etc/trafficserver/records.config ${ATS_DIR}/etc/trafficserver/records.config.bak
    cp -n ${ATS_DIR}/etc/trafficserver/remap.config ${ATS_DIR}/etc/trafficserver/remap.config.bak
    cp -n ${ATS_DIR}/etc/trafficserver/cache.config ${ATS_DIR}/etc/trafficserver/cache.config.bak
    cp -n ${ATS_DIR}/etc/trafficserver/storage.config ${ATS_DIR}/etc/trafficserver/storage.config.bak
    cp -n ${ATS_DIR}/etc/trafficserver/ip_allow.config ${ATS_DIR}/etc/trafficserver/ip_allow.config.bak
    cp -n ${ATS_DIR}/etc/trafficserver/plugin.config ${ATS_DIR}/etc/trafficserver/plugin.config.bak
    cp ${ATS_DIR}/etc/trafficserver/records.config.bak ${ATS_DIR}/etc/trafficserver/records.config
    cp ${ATS_DIR}/etc/trafficserver/remap.config.bak ${ATS_DIR}/etc/trafficserver/remap.config
    cp ${ATS_DIR}/etc/trafficserver/cache.config.bak ${ATS_DIR}/etc/trafficserver/cache.config
    cp ${ATS_DIR}/etc/trafficserver/storage.config.bak ${ATS_DIR}/etc/trafficserver/storage.config
    cp ${ATS_DIR}/etc/trafficserver/ip_allow.config.bak ${ATS_DIR}/etc/trafficserver/ip_allow.config
    cp ${ATS_DIR}/etc/trafficserver/plugin.config.bak ${ATS_DIR}/etc/trafficserver/plugin.config
    # echo "$(date +%H:%M:%S) ATS config done backup" >> ${status_log}
    echo "$(date) ATS config done backup" >> ${status_log}


    # echo "$(date +%H:%M:%S) ATS config setup done" >> ${status_log}
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}


stopApp() {
    echo -ne stop >/tmp/c2dn/monitorCmd 2>/dev/null || true
    killall -9 client 2>/dev/null || true;
    killall -9 origin 2>/dev/null || true;
    killall -9 frontend 2>/dev/null || true; 
    killall -9 main 2>/dev/null || true; 
    killall -9 node_exporter 2>/dev/null || true;
    sleep 2;
    killall -9 python3 2>/dev/null || true;
}

stopOrigin() {
    stopApp
}

stopClient() {
    stopApp
}

stopCDN() {
    [ -f ${ATS_DIR}/bin/trafficserver ] && ${ATS_DIR}/bin/trafficserver stop 2>&1 >/dev/null; sleep 8; 
    pkill traffic_manager || true; pkill traffic_server || true;
    udevadm trigger --subsystem-match=block;
    # clear ATS disk cache;
    # ${ATS_DIR}/bin/traffic_server -Cclear >/dev/null; sleep 2;
    rm ${ATS_DIR}/var/log/trafficserver/diags.log 2>/dev/null || true;
    rm ${ATS_DIR}/var/log/trafficserver/squid.blog 2>/dev/null || true;
    rm ${ATS_DIR}/var/trafficserver/records.snap 2>/dev/null || true; 

    stopApp
}

buildApp() {
    pushd ${C2DN_DIR} > /dev/null;
    rm client origin frontend main 2>/dev/null || true;
    # /usr/local/go/bin/go build src/main/main.go > /dev/null;
    /usr/local/go/bin/go build ./cmd/client > /dev/null;
    /usr/local/go/bin/go build ./cmd/origin > /dev/null;
    /usr/local/go/bin/go build ./cmd/frontend > /dev/null;
    popd > /dev/null

    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}


startLocalPyOrigin() {
    pkill gunicorn 2>/dev/null || true;
    cd ${EXP_DIR}/PyOrigin/;
    rm -r origin.traffic 2>/dev/null || true;
    rm origin.py
    # don't --preload, otherwise traffic file will be only one
    screen -S gunicorn -dm /usr/local/bin/gunicorn -w 16 -b 0.0.0.0:2048 -k eventlet --preload --keep-alive 30 --access-logfile /tmp/origin.log --error-logfile /tmp/origin.logerr --log-level info --capture-output --worker-connections 32000 localOrigin:app;   #  --threads 4 --daemon
    sleep 2;
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}

startLocalOrigin() {
    debug "start local origin"
    pushd ${C2DN_DIR} > /dev/null;
    screen -S origin -L -Logfile /tmp/c2dn/screen/origin -dm ./origin;
    sleep 8; # origin needs to allocate large data when starts, change to 2 is too small
    popd > /dev/null
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}

# startOrigin(){
#     pkill gunicorn 2>/dev/null || true;
#     cd $HOME/CDN/C2DN_GO/PyOrigin/;
#     rm -r origin.traffic 2>/dev/null || true;
#     rm localOrigin.py;
#     screen -S gunicorn -dm /usr/local/bin/gunicorn -w 64 -b 0.0.0.0:2048 -k eventlet --preload --keep-alive 30 --access-logfile /tmp/origin.log --error-logfile /tmp/origin.logerr --log-level info --capture-output --worker-connections 32000 origin:app;   #  --daemon
#     sleep 2;
# }

checkResponse() {
    url=$1
    correct_resp=$2

    # resp=$(curl -s -H "Bucket:1" ${url} || true)
    resp=$(curl -s ${url} || true)
    if [[ "$resp" != *"${correct_resp}"* ]]; then
        error "check response curl -s ${url}: should be \"${correct_resp}\" get \"${resp}\""
    fi
}

checkLocalOrigin() {
    debug "check local origin"
    checkResponse "http://127.0.0.1:${origin_port}" "Hello, world! This is origin"
    checkResponse '-H "objType:chunk" -H "ecChunk:4_3_0" http://127.0.0.1:'${origin_port}'/akamai/ab_24' '********'
    info 'check origin pass'

    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}

startRemoteClient() {
    debug "start remote client"
    rm ${C2DN_DIR}/client.status 2>/dev/null || true

    pushd ${C2DN_DIR} >/dev/null;
    speedup=$1
    randomRoute=$2

    debug "./client -mode=replayOpenloop -trace=${LOCAL_TRACE_DIR}/akamai.bin -unavailTrace=${unavail_trace} -replayStartTs=${eval_start_ts} -replayEndTs=${eval_end_ts} -ignoreRemoteReq=0 -uniqueObj=0 -remoteOrigin=${use_remote_origin} -randomRoute=1 -clientID=-1 -concurrency=32 -replaySpeedup=${speedup} -nServers=10 ${cdn_pub_ip_port_str}"
    screen -S replay -L -Logfile /tmp/c2dn/screen/replay_${eval_start_ts}_${eval_end_ts} -dm ./client -mode=replayOpenloop -trace=${LOCAL_TRACE_DIR}/akamai.bin -unavailTrace=${unavail_trace} -replayStartTs=${eval_start_ts} -replayEndTs=${eval_end_ts} -ignoreRemoteReq=0 -uniqueObj=0 -remoteOrigin=${use_remote_origin} -randomRoute=${randomRoute} -clientID=-1 -concurrency=32 -replaySpeedup=${speedup} -nServers=10 ${cdn_pub_ip_port_str}

    popd >/dev/null; 
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}

startCDN() {
    startLocalOrigin

    ${ATS_DIR}/bin/trafficserver start 2>&1 >/dev/null;
    sleep 20

    pushd ${C2DN_DIR} >/dev/null;
    debug "./frontend -mode=${mode} -repFactor=${rep_factor} -EC_n=${EC_n} -EC_k=${EC_k} -ramCacheSize=${fe_ram_size} -nServers=${#ats_priv_ip_port[@]} -unavailTrace=${unavail_trace} -nodeIdx=${nodeIdx} ${ats_priv_ip_port_str}"
    screen -L -Logfile /tmp/c2dn/screen/frontend -S frontend -dm ./frontend -mode=${mode} -repFactor=${rep_factor} -EC_n=${EC_n} -EC_k=${EC_k} -ramCacheSize=${fe_ram_size} -nServers=${#ats_priv_ip_port[@]} -unavailTrace=${unavail_trace} -nodeIdx=${nodeIdx} ${ats_priv_ip_port_str};


    # if [[ "${system}" == "C2DN" ]]; then
    #     debug "./frontend -mode=C2DN -EC_n=${EC_n} -EC_k=${EC_k} -ramCacheSize=${fe_ram_size} -nServers=${#ats_priv_ip_port[@]} -unavailTrace=${unavail_trace} -nodeIdx=${nodeIdx} ${ats_priv_ip_port_str}"
    #     screen -L -Logfile /tmp/c2dn/screen/frontend -S frontend -dm ./frontend -mode=C2DN -EC_n=${EC_n} -EC_k=${EC_k} -ramCacheSize=${fe_ram_size} -nServers=${#ats_priv_ip_port[@]} -unavailTrace=${unavail_trace} -nodeIdx=${nodeIdx} ${ats_priv_ip_port_str};

    #     debug "./frontend -mode=naiveCoding -EC_n=${EC_n} -EC_k=${EC_k} -ramCacheSize=${fe_ram_size} -nServers=${#ats_priv_ip_port[@]} -unavailTrace=${unavail_trace} -nodeIdx=${nodeIdx} ${ats_priv_ip_port_str}"
    #     screen -L -Logfile /tmp/c2dn/screen/frontend -S frontend -dm ./frontend -mode=naiveCoding -EC_n=${EC_n} -EC_k=${EC_k} -ramCacheSize=${fe_ram_size} -nServers=${#ats_priv_ip_port[@]} -unavailTrace=${unavail_trace} -nodeIdx=${nodeIdx} ${ats_priv_ip_port_str};

    #     debug "./frontend -mode=noRep -EC_n=${EC_n} -EC_k=${EC_k} -ramCacheSize=${fe_ram_size} -nServers=${#ats_priv_ip_port[@]} -unavailTrace=${unavail_trace} -nodeIdx=${nodeIdx} ${ats_priv_ip_port_str}"
    #     screen -L -Logfile /tmp/c2dn/screen/frontend -S frontend -dm ./frontend -mode=noRep -repFactor=1 -EC_n=${EC_n} -EC_k=${EC_k} -ramCacheSize=${fe_ram_size} -nServers=${#ats_priv_ip_port[@]} -unavailTrace=${unavail_trace} -nodeIdx=${nodeIdx} ${ats_priv_ip_port_str};

    #     # warn "use local only ATS"
    #     # screen -S frontend -dm ./frontend -EC_n=${EC_n} -EC_k=${EC_k} -ramCacheSize=${fe_ram_size} -nServers=10 127.0.0.1:${ats_port} 127.0.0.1:${ats_port} 127.0.0.1:${ats_port} 127.0.0.1:${ats_port} 127.0.0.1:${ats_port} 127.0.0.1:${ats_port} 127.0.0.1:${ats_port} 127.0.0.1:${ats_port} 127.0.0.1:${ats_port} 127.0.0.1:${ats_port};

    #     # warn "use Origin as Cache, REMEMBER TO CHANGE THIS"
    #     # screen -S frontend -dm ./frontend -EC_n=${EC_n} -EC_k=${EC_k} -ramCacheSize=${fe_ram_size} -nServers=1 127.0.0.1:2048
    # elif [[ "${system}" == "CDN" ]]; then
    #     debug "./frontend -mode=twoRepAlways -EC_n=${EC_n} -EC_k=${EC_k} -ramCacheSize=${fe_ram_size} -nServers=${#ats_priv_ip_port[@]} -unavailTrace=${unavail_trace} -nodeIdx=${nodeIdx} ${ats_priv_ip_port_str}"
    #     screen -L -Logfile /tmp/c2dn/screen/frontend -S frontend -dm ./frontend -mode=twoRepAlways -EC_n=${EC_n} -EC_k=${EC_k} -ramCacheSize=${fe_ram_size} -nServers=${#ats_priv_ip_port[@]} -unavailTrace=${unavail_trace} -nodeIdx=${nodeIdx} ${ats_priv_ip_port_str};
    # fi

    sleep 20    # this is needed for ATS to initialize disk (or load data from disk)
    # because C2DN pushes chunks to ATS, so we should ask ATS not store the objects from origin 
    # if [[ "${system}" == "C2DN" ]]; then
    curl -s "http://127.0.0.1:${origin_port}/setNoCache/" >/dev/null; 
    # fi

    popd >/dev/null; 
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}

checkCDN() {
    checkResponse "http://127.0.0.1:${origin_port}" "Hello, world! This is origin"
    checkResponse '-H "objType:chunk" -H "ecChunk:4_3_0" http://127.0.0.1:'${origin_port}'/akamai/ab_24' '********'
    debug 'check local origin pass'

    checkResponse "http://127.0.0.1:${ats_port}" "Hello, world! This is origin"
    checkResponse "http://127.0.0.1:${ats_port}/akamai/ab_24" '************************'
    debug 'check ats pass'

    checkResponse "http://127.0.0.1:${fe_port}" "Hello, world I am Frontend!"
    checkResponse '-H Bucket:1 http://127.0.0.1:'${fe_port}'/akamai/ab_24' '************************'
    debug 'check frontend pass'

    info 'check CDN response pass'
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}

startMonitoring() {
    script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
    pushd ${script_dir} >/dev/null 
    echo -ne run >/tmp/c2dn/monitorCmd;
    screen -S monitor -L -Logfile /tmp/c2dn/screen/monitor -dm python3 monitor.py;

    cd ${EXP_DIR}; 
    screen -S monitor -L -Logfile /tmp/c2dn/screen/nodeExporter -dm ./node_exporter --web.disable-exporter-metrics --web.listen-address="0.0.0.0:9100" 

    popd >/dev/null
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}
}

clearPrevExp() {
    pushd ${C2DN_DIR} >/dev/null
    sudo rm -f /tmp/log* /tmp/c2dn/* 2>/dev/null || true;
    rm -f ${ATS_DIR}/var/log/trafficserver/* 2>/dev/null || true
    rm -r client.status *latency* 2>/dev/null || true;
    [ -f ${ATS_DIR}/bin/trafficserver ] && ${ATS_DIR}/bin/trafficserver stop 2>&1 >/dev/null; sleep 8; 
    info "disable ATS cache clear"
    [ -f ${ATS_DIR}/bin/trafficserver ] && ${ATS_DIR}/bin/traffic_server -Cclear 2>&1 >/dev/null; sleep 8; 

    popd >/dev/null
    echo $(date) ${FUNCNAME[0]} done >> ${status_log}       
}

downloadTraceClient() {
    aws s3 cp ${trace_path} ${LOCAL_TRACE_DIR}/akamai.bin;

    # aws s3 cp ${eval_trace_src}.remote ${LOCAL_TRACE_DIR}/akamai.bin.eval;
}

downloadTraceCDN() {
    aws s3 cp ${trace_path} ${LOCAL_TRACE_DIR}/akamai.bin;

    # aws s3 cp ${warmup_trace_src}.${nodeIdx} ${LOCAL_TRACE_DIR}/akamai.bin.warmup;
    # aws s3 cp ${eval_trace_src}.${nodeIdx} ${LOCAL_TRACE_DIR}/akamai.bin.eval;
    # aws s3 cp ${eval_thrpt_trace_src}.${nodeIdx} ${LOCAL_TRACE_DIR}/akamai.bin.eval.thrpt;
    # aws s3 cp ${warmup_uniq_ram_trace_src}.${nodeIdx} ${LOCAL_TRACE_DIR}/akamai.bin.warmup.uniq.ram;
    # aws s3 cp ${warmup_uniq_disk_trace_src}.${nodeIdx} ${LOCAL_TRACE_DIR}/akamai.bin.warmup.uniq.disk;


    # debug "aws s3 cp ${warmup_trace_src}.${nodeIdx} ${LOCAL_TRACE_DIR}/akamai.bin.warmup;"
    # debug "aws s3 cp ${eval_trace_src}.${nodeIdx} ${LOCAL_TRACE_DIR}/akamai.bin.eval;"
    # debug "aws s3 cp ${eval_thrpt_trace_src}.${nodeIdx} ${LOCAL_TRACE_DIR}/akamai.bin.eval.thrpt;"
    # debug "aws s3 cp ${warmup_uniq_ram_trace_src}.${nodeIdx} ${LOCAL_TRACE_DIR}/akamai.bin.warmup.uniq.ram;"
    # debug "aws s3 cp ${warmup_uniq_disk_trace_src}.${nodeIdx} ${LOCAL_TRACE_DIR}/akamai.bin.warmup.uniq.disk;"
}

export -f setHostname
export -f initDir
export -f initSettings
export -f synctime
export -f prepareDisk
export -f performanceTuning

export -f installPackages
export -f installATS
export -f installISAL
export -f installGO


export -f stopApp
export -f stopOrigin
export -f stopCDN
export -f stopClient

export -f buildApp
export -f setupATS

export -f startLocalOrigin
export -f checkLocalOrigin
export -f startMonitoring


export -f startCDN
export -f checkCDN
export -f startRemoteClient
export -f clearPrevExp

