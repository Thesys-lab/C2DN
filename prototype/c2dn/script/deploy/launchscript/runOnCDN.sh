#!/usr/bin/env bash


export expname=$1
# set -x

user=ubuntu
source ../scriptcore/op.sh
source ./params.sh
source ./ip.sh


tmp() {
    stopExp
    collectResult
}

checkTTop(){
    for ip in ${cdn_pub_ips[*]}; do
        # echo -n -e $ip '\t';
        ssh -o StrictHostKeyChecking=no ${user}@$ip -q -t '''echo -ne $(hostname)
            echo -ne '${ip}': "\t"
            echo -n "origin: ";
            curl 127.0.0.1:'${origin_port}'/akamai/ab_25; echo -n ", "
            echo -n "ats: ";
            curl 127.0.0.1:'${ats_port}'/akamai/ab_25; echo -n ", "
            echo -n "frontend: "; curl 127.0.0.1:'${fe_port}'/akamai/ab_25; echo

            echo q | $HOME/CDN/ATSRelease/bin/traffic_top | aha --line-fix | html2text -width 999 | grep -e Disk;
            # echo q | $HOME/CDN/ATSRelease/bin/traffic_top | aha --line-fix | html2text -width 999 |grep Disk;
            sudo iotop -o -P -b -n 1 -q|grep DISK | grep -v PID;

        '''
    done
}

tailStat(){
    for ip in ${cdn_pub_ips[*]}; do
        echo -n -e $ip '\t'
        ssh -o StrictHostKeyChecking=no ubuntu@$ip -q -t '''echo $(hostname)
            tail -n 1 /tmp/c2dn/output/client.stat 

        '''
    done
}

getdstat() {
    for ip in ${cdn_pub_ips[*]}; do
        echo -n -e $ip '\t';
        ssh ${user}@$ip -q -t '''echo $(hostname)
            /usr/bin/dstat -D total,nvme0n1,nvme1n1,nvme2n1 -d --disk-util --disk-tps --disk-avgqu --disk-wait --integer --net --top-cpu 5 2 
            # tail -n 20 /home/ubuntu/CDN/ATSRelease/var/log/trafficserver/diags.log
        '''
        echo '**************************************************************************************************************'
    done
}

grepPanic(){
    for ip in ${cdn_pub_ips[*]}; do
        echo -n -e $ip '\t';
        ssh -o StrictHostKeyChecking=no ubuntu@$ip -q -t '''echo $(hostname)
            grep PANIC /tmp/c2dn/log/*
        '''
        echo '**************************************************************************************************************'
    done
}

tailLog(){
    for ip in ${cdn_pub_ips[*]}; do
        echo -n -e $ip '\t';
        ssh ${user}@$ip -q -t '''echo $(hostname)
            tail -n 20 /tmp/c2dn/log/frontend
            # tail -n 20 /home/ubuntu/CDN/ATSRelease/var/log/trafficserver/diags.log
        '''
        echo '**************************************************************************************************************'
    done
}

mvBackup() {
    for ip in ${cdn_pub_ips[*]}; do
        echo -n -e $ip '\t';
        ssh -o StrictHostKeyChecking=no ubuntu@$ip -q -t '''echo $(hostname)
            mv /disk0/backup2/cache.db /disk0/backup2/aws_C2DN_akamai2_expLatency_unavail0_43_1000G.db
            mv /disk1/backup2/cache.db /disk1/backup2/aws_C2DN_akamai2_expLatency_unavail0_43_1000G.db
        '''
        echo '**************************************************************************************************************'
    done
}

grepATSError(){
    for ip in ${cdn_pub_ips[*]}; do
        echo -n -e $ip '\t';
        ssh -o StrictHostKeyChecking=no ubuntu@$ip -q -t '''echo $(hostname)
            ls /home/ubuntu/CDN/ATSRelease/var/log/trafficserver/
            if [ -f /home/ubuntu/CDN/ATSRelease/var/log/trafficserver/error.log ]; then
                cat /home/ubuntu/CDN/ATSRelease/var/log/trafficserver/error.log
            fi
            # grep FATAL /tmp/*log*
            # grep WARN /tmp/*log*
        '''
        echo '**************************************************************************************************************'
    done
}


monitorFEMetrics() {
    for i in `seq 0 $((${#cdn_pub_ips[@]}-1))`; do
        ip="${cdn_pub_ips[i]}"
        (echo -n -e $ip '\t' ;
        ssh -q ${user}@$ip -tt '''echo $(hostname)
            curl -s 127.0.0.1:2022/metrics | grep ^frontend > '${ip}'.fe.metrics || true
            curl -s 127.0.0.1:2022/metrics | grep ^frontend | grep -e nReq -e err -e traffic -e chunk || true
        ''')
    done 
}


checkCacheStable() {
    # export IFS=$' '
    DAT_DIR=temp/BMR/
    mkdir -p ${DAT_DIR} 2>/dev/null

    nCDN=$((${#cdn_pub_ips[@]}-1))
    for i in `seq 0 $nCDN`; do
        (   ip="${cdn_pub_ips[i]}"
            echo -e $i: $ip '\t';


        scp -i ~/.ssh/id_rsa -o StrictHostKeyChecking=no -q -r ubuntu@$ip:/home/ubuntu/CDN/C2DN_GO/origin.traffic ${DAT_DIR}/origin.traffic.$i
        scp -i ~/.ssh/id_rsa -o StrictHostKeyChecking=no -q -r ubuntu@$ip:/home/ubuntu/CDN/C2DN_GO/client.stat ${DAT_DIR}/client.stat.$i
        ) &

    done
    wait
    # python3 plotBMR.py
}

pullIfstat() {
    DAT_DIR=temp/ifstat/
    mkdir -p ${DAT_DIR} 2>/dev/null

    nCDN=$((${#cdn_pub_ips[@]}-1))
    for i in `seq 0 $nCDN`; do
        (   ip="${cdn_pub_ips[i]}"
            echo -e $i: $ip '\t';
            scp -i ~/.ssh/id_rsa -o StrictHostKeyChecking=no -q -r ubuntu@$ip:/home/ubuntu/ifstat.log ${DAT_DIR}/ifstat.log.$i
        ) &
    done
    wait
    python3 logprocessing.py
}


checkFEStat(){
    nCDN=$((${#cdn_pub_ips[@]}-1))
    for i in `seq 0 $nCDN`; do
        (   ip="${cdn_pub_ips[i]}"
            echo -e -n $i: $ip '\t';
            ssh -o StrictHostKeyChecking=no ${user}@$ip -q -t '''echo $(hostname)
                curl 127.0.0.1:8081/stat/
            '''
        )
    done
}


createImage() {
    nCDN=$((${#cdn_pub_ips[@]}-1))
    for i in `seq 0 $nCDN`; do
        (   ip="${cdn_pub_ips[i]}"
            echo -e -n $i: $ip '\t';
            ssh -o StrictHostKeyChecking=no ubuntu@$ip -q -t """
                sudo mount /dev/nvme0n1p2 /home/ubuntu/disk1
                sudo mount /dev/nvme1n1p2 /home/ubuntu/disk2
                rm /home/ubuntu/disk1/* 2>/dev/null
                rm /home/ubuntu/disk2/* 2>/dev/null
                sudo chown -R ubuntu /home/ubuntu/disk1
                sudo chown -R ubuntu /home/ubuntu/disk2
                sudo dd if=/dev/nvme0n1p1 bs=64K status=progress of=/home/ubuntu/disk1/nvme0n1p1.img.$i
                aws s3 cp /home/ubuntu/disk1/nvme0n1p1.img.$i s3://jasondatavaluable/C2DN/disk/${expname}/nvme0n1p1.img.$i &
                sudo dd if=/dev/nvme1n1p1 bs=64K status=progress of=/home/ubuntu/disk2/nvme1n1p1.img.$i
                aws s3 cp /home/ubuntu/disk2/nvme1n1p1.img.$i s3://jasondatavaluable/C2DN/disk/${expname}/nvme1n1p1.img.$i
                wait
            """
        ) &
    done
    wait
}

updateFE() {
    mydir=$(pwd);
    # cd ../../;
    # tar --exclude='.git' --exclude='pkg' --exclude='plot' --exclude='bin' --exclude='build' --exclude='doc' -zcf C2DN_GO.tar.gz C2DN_GO;
    # tar --exclude='.git' --exclude='pkg' --exclude='plot' --exclude='bin' --exclude='build' --exclude='doc' --exclude 'github.com' --exclude 'golang.org' --exclude 'go.uber.org' -zcf C2DN_GO.tar.gz C2DN_GO;
    # mv C2DN_GO.tar.gz $mydir;
    # cd $mydir;
    # echo '################################### localPreprocessing finished ####################################'


    # cd ../../C2DN_GO/;
    nCDN=$((${#cdn_pub_ips[@]}-1))
    for i in `seq 0 $nCDN`; do
        (ip="${cdn_pub_ips[i]}"
        echo -e $i: $ip '\t';
        rsync -r "${top_level_dir}../c2dn" --exclude .git --exclude github.com --exclude *.org ${user}@$ip:CDN/
        # scp -i ~/.ssh/id_rsa -o StrictHostKeyChecking=no -q C2DN_GO.tar.gz ubuntu@$ip:/home/ubuntu/
        ssh -o StrictHostKeyChecking=no ${user}@$ip -tt '''
            # rm -rf CDN software
            echo $(hostname)
            rm -r /tmp/log* 2>/dev/null;
            killall -9 frontend 2>/dev/null; killall -9 client 2>/dev/null; killall -9 origin 2>/dev/null;
            export PATH=$PATH:/usr/local/go/bin; export GOPATH=$HOME/CDN/C2DN_GO; export GOBIN=$HOME/CDN/C2DN_GO;
            rm main client frontend origin 2>/dev/null; /usr/local/go/bin/go install main;
            cd CDN/C2DN_GO/;
            cp main frontend; cp main client; cp main origin;
            screen -S origin -dm ./main -role=origin;
        '''
        ) &
    done
    wait

    # cd $mydir;
    # checkAlive
}


d(){
    nCDN=$((${#cdn_pub_ips[@]}-1))
    for i in `seq 0 $nCDN`; do
        ip="${cdn_pub_ips[i]}"
        echo -e $i: $ip '\t';
        (ssh -o StrictHostKeyChecking=no ubuntu@$ip -q -tt '''
            sudo reboot;
        ''')&
    done
    wait
}


test(){
    for ip in ${cdn_pub_ips[*]}; do
        echo $ip;
        (ssh -o StrictHostKeyChecking=no ${user}@$ip -q -t '''echo $(hostname)
            df -h
        ''')
    done
    wait
}


up() {
    nCDN=$((${#cdn_pub_ips[@]}-1))
    for i in `seq 0 $nCDN`; do
        ip="${cdn_pub_ips[i]}"
        (ssh -o StrictHostKeyChecking=no ${user}@$ip -q -tt '''
            # aws s3 cp /run/shm/C2DN/stat/procStat s3://jasondatadeletable/C2DN/prototype/$(date +%Y-%m-%d)/'${expname}'/RR'${REQUEST_RATE}'/stat/procStat.'${i}'
            # aws s3 cp /run/shm/C2DN/stat/sysStat s3://jasondatadeletable/C2DN/prototype/$(date +%Y-%m-%d)/'${expname}'/RR'${REQUEST_RATE}'/stat/sysStat.'${i}'
            # echo $(date +%H:%M:%S) ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>" end >> /run/shm/C2DN/stat/ifconfig.log
            # ifconfig >> /run/shm/C2DN/stat/ifconfig.log
            # aws s3 cp /run/shm/C2DN/stat/ifconfig.log s3://jasondatadeletable/C2DN/prototype/$(date +%Y-%m-%d)/'${expname}'/RR'${REQUEST_RATE}'/stat/ifconfig2.'${i}'
            # cat /run/shm/C2DN/stat/ifconfig.log
            # cat $HOME/ifconfig
            aws s3 cp --quiet /run/shm/C2DN/stat/procStat s3://jasondatadeletable/C2DN/prototype/$(date +%Y-%m-%d)/'${expname}'/RR'${REQUEST_RATE}'/stat/procStat.'${i}'
            # aws s3 cp --quiet /run/shm/C2DN/stat/sysStat s3://jasondatadeletable/C2DN/prototype/$(date +%Y-%m-%d)/'${expname}'/RR'${REQUEST_RATE}'/stat/sysStat2.'${i}'
        ''') &
    done
    wait
}


eval $2
# startClient



