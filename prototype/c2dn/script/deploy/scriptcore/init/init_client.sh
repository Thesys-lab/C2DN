#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source ${DIR}/init_utils.sh
source ${DIR}/init_func.sh


init=0
build=0
start=0
stop=0
expname=""
trace=""
speedup=1
randomRoute=1


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
            speedup=$2
            randomRoute=${3:-1}
            shift
            ;;
        "-d" | "--stop")
            stop=1
            ;;
        "-t" | "--trace")
            shift
            trace=$1
            ;;
    esac
    shift
done

initDir

if [ $stop = 1 ]; then 
    info "stop client"
    stopClient
fi

if [ $init = 1 ]; then
    info "init client"
    initSettings
    setHostname client-${expname}
    installPackages
    synctime
    performanceTuning
    installGO
fi 

if [ $build = 1 ]; then 
    info "build client"
    clearPrevExp
    buildApp
    downloadTraceClient
fi

if [ $start = 1 ]; then
    info "start remote client"
    startMonitoring
    startRemoteClient ${speedup} ${randomRoute}
fi    



