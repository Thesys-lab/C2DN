#!/usr/bin/env bash 
# if region is not east-1, need to update this script
# HOME=/home/ubuntu/
# TS_ROOT=/home/ubuntu/CDN/ATSRelease/


DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source ${DIR}/init_utils.sh
source ${DIR}/init_func.sh


usage() {
    echo "Usage: $0 -e expname -i[nitialize] -s[tart]"
}



usage() {
    echo "Usage: $0 -e expname -i[nitialize] -b[uild] -s[tart] -d[stop]"
}



init=0
build=0
start=0
stop=0
# nodeIdx=999999
expname=""


while [ "${1:-}" != "" ]; do
    case "$1" in
        "-e" | "--exp" | "--expname")
            shift
            expname=$1
            ;;
        "-n" | "--nodeIdx")
            shift
            nodeIdx=$1
            ;;
        "-i" | "--init" | "--initialize")
            init=1
            ;;
        "-b" | "--build")
            build=1
            ;;
        "-s" | "--start")
            start=1
            ;;
        "-d" | "--stop")
            stop=1
            ;;
    esac
    shift
done


# [ ${nodeIdx} = 999999 ] && echo "init_cdn does not receive nodeIdx" && exit 1


initDir

if [ $stop = 1 ]; then 
    info "stop CDN server"
    stopCDN
fi 

if [ $init = 1 ]; then
    info "init CDN server"
    rm ${status_log} 2>/dev/null || true
    initSettings
    setHostname CDN-${expname}-${nodeIdx}
    installPackages
    synctime

    performanceTuning
    installATS
    # installISAL
    installGO

    # this needs to be after ATS installed 
    [ $testbed = "aws" ] && prepareDiskAWS; 
    downloadTraceCDN
fi 

if [ $build = 1 ]; then 
    info "build CDN server"
    clearPrevExp
    setupATS 
    buildApp
fi



if [ $start = 1 ]; then
    info "start CDN server"
    # startMonitoring
    startCDN
    checkCDN 
fi    



