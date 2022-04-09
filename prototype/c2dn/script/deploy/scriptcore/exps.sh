#!/bin/bash
# DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
import() {
    local curr_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
    source "${curr_dir}/const.sh"
    source "${launch_script_dir}/params.sh"
    source "${scriptcore_dir}/utils.sh"
    source "${scriptcore_dir}/setup.sh"
    source "${scriptcore_dir}/op.sh"
    # source "$scriptcore_dir/common.sh"
}

import


expLatency(){
    info "run latency experiment"

    checkBeforeRun
    localProcessing
    prepareRemoteOrigin
    prepareCDN

    checkNodeStatus "cluster setup finished"
    askContinue "cluster ready, next is warmup for latency exp"

    if [[ ${use_backuped_disk_cache} == 1 ]]; then restoreDiskCache; askContinue "disk cache have been restored"; fi
    warmupCluster 
    # if [[ ${use_backuped_disk_cache} == 0 ]]; then backupDiskCache; askContinue "disk cache have been backup"; fi
    
    prepareRemoteClient
    askContinue "client ready, next is to run latency exp"

    prepareForEval
    runLocalClient 1
    runRemoteClient 1
    waitForEval 
    stopExp
    collectResult    

    # echo $(date +%H:%M:%S) "now terminating VMs"
    # python3 $DIR/launchVMs.py terminateVMsIP ${availability_zone} ${cdn_pub_ips[@]}
}


expMacrobenchmark(){
    echo $(date +%H:%M:%S) $SEPARATOR run macro_benchmark experiment $SEPARATOR

    checkBeforeRun
    localProcessing
    prepareRemoteOrigin
    prepareCDN

    uploadC2DNApp
    initCDN
    prepareCDN_original
    checkNodeStatus "cluster setup finished"

    askContinue "cluster setup finished next is warmup for latency exp"
    warmupCluster "supportBucket"        # restored disk cache will have RAM only warmup

    prepareRemoteClient ${eval_trace_src}
    askContinue "client started, next is to run latency exp"

    akamaiLatencyExp ${eval_trace_src}

    # echo $(date +%H:%M:%S) "now terminating VMs"
    # python3 $DIR/launchVMs.py terminateVMsIP ${availability_zone} ${cdn_pub_ips[@]}
}


expThrpt(){
    echo $(date +%H:%M:%S) $SEPARATOR run throughput experiment $SEPARATOR
    export NO_CHECK=1
    checkBeforeRun
    localProcessing
    prepareRemoteOrigin
    prepareCDN

    uploadC2DNApp
    initCDN

    for speedup in 2 3 4 5; do
        export REPLAY_SPEEDUP=$speedup
        echo $(date +%H:%M:%S) $SEPARATOR $expname now begin preparing throughput exp speedup $REPLAY_SPEEDUP $SEPARATOR

        if [[ ${USE_STORED_DISKCACHE} == 1 ]]; then restoreDiskCache; fi
        prepareCDN_original
        checkNodeStatus "cluster setup finished "

        askContinue "cluster setup finished next is warmup for latency exp"
        warmupCluster "supportBucket"        # restored disk cache will have RAM only warmup

        prepareRemoteClient ${eval_trace_src}
        askContinue "client started, next is to run latency exp"

        akamaiLatencyExp ${eval_trace_src}

        echo $(date +%H:%M:%S) $SEPARATOR $expname speedup $REPLAY_SPEEDUP evaluation finishes $SEPARATOR
    done

    # echo "$(date +%H:%M:%S) now terminating VMs"
    # python3 $DIR/launchVMs.py terminateVMsIP ${availability_zone} ${cdn_pub_ips[@]}
}


expUnavailability(){
    echo $(date +%H:%M:%S) $SEPARATOR run unavailability experiment $SEPARATOR
    export unavail_trace=unavailability.one

    checkBeforeRun
    localProcessing
    prepareRemoteOrigin
    prepareCDN

    uploadC2DNApp
    # initCDN
    # if [[ ${USE_STORED_DISKCACHE} == 1 ]]; then restoreDiskCache; askContinue "disk cache have downloaded"; fi
    prepareCDN_original
    checkNodeStatus "cluster setup finished"

    askContinue "cluster setup finished next is warmup for unavailability exp"
    warmupCluster "supportBucket"        # restored disk cache will have RAM only warmup

    prepareRemoteClient ${eval_trace_src}
    askContinue "client started, next is to run unavailability exp"

    akamaiLatencyExp ${eval_trace_src}

    # echo $(date +%H:%M:%S) "now terminating VMs"
    # python3 $DIR/launchVMs.py terminateVMsIP ${availability_zone} ${cdn_pub_ips[@]}
}


