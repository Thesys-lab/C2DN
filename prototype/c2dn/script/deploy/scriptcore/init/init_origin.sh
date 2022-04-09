#!/bin/bash


DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source ${DIR}/init_utils.sh
source ${DIR}/init_func.sh


usage() {
    echo "Usage: $0 -e expname -i[nitialize] -b[uild] -s[tart] -d[stop]"
}



init=0
build=0
start=0
stop=0
expname=""


while [ "${1:-}" != "" ]; do
    case "$1" in
        "-e" | "--exp" | "--expname")
            shift
            expname=$1
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


initDir

if [ $stop = 1 ]; then 
    info "stop origin"
    stopOrigin
fi 

if [ $init = 1 ]; then
    info "init origin"
    initSettings
    setHostname origin-${expname}
    installPackages
    synctime
    performanceTuning
    # installISAL
    installGO    
fi 

if [ $build = 1 ]; then 
    info "build origin"
    clearPrevExp
    buildApp
fi

if [ $start = 1 ]; then
    info "start origin"
    startLocalOrigin
    checkLocalOrigin
    startMonitoring
fi

# exit 1

