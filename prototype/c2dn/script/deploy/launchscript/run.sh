#!/usr/bin/env bash
pushd "$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)" >/dev/null || return 
source "../scriptcore/const.sh"


trap 'previous_command=$this_command; this_command=$BASH_COMMAND' DEBUG
trap 'echo "exit status $? due to command \"$previous_command\"" and \"$this_command\"' EXIT
trap 'echo error at line ${LINENO}' ERR


# set -x


# let script exit if a command fails
set -o errexit
#let script exit if an unsed variable is used
set -o nounset




usage() { 
    echo "Usage: $0 testbed system expSetup exp 
        testbed:    aws/cloudlab 
        system:     CDN/C2DN 
        expSetup:   akamai1/akamai2/akamai1Unavailability/akamai2Unavailability/macrobenchmark
        exp:        expLatency/expThrpt/expUnavailability

        for example: ./run.sh -t aws -s CDN -w akamai2 -e expLatency
        " 1>&2; 
    exit 0
}


CDN(){
    export EC_n=2
    export EC_k=1
    export mode="twoRepAlways"
    export rep_factor=2

    export placement_group=ats
    export availability_zone="us-east-1a"
    export expname=${expname_BASE}_${server_cache_size}G
}

C2DN(){
    export EC_n=4
    export EC_k=3
    export mode="C2DN"
    export rep_factor=2

    export placement_group=c2dn1
    export availability_zone="us-east-1b"
    export expname=${expname_BASE}_${EC_n}${EC_k}_${server_cache_size}G
}

noRep() {
    export EC_n=1
    export EC_k=1
    export mode="noRep"
    export rep_factor=1

    export placement_group=c2dn2
    export availability_zone="us-east-1c"
    export expname=${expname_BASE}_${server_cache_size}G
}

naiveCoding() {
    export EC_n=4
    export EC_k=3
    export mode="naiveCoding"
    export rep_factor=2

    export placement_group=c2dn3
    export availability_zone="us-east-1a"
    export expname=${expname_BASE}_${EC_n}${EC_k}_${server_cache_size}G
}

akamai1(){
    export trace_type=akamai1
    export server_cache_size=100
    export ats_ram_size=209715200
    export fe_ram_size=8589934592
    # export unavail_trace=config/unavailability.one
    export trace_path="s3://juncheng-data/C2DN/akamai.bin.scale10"
}

akamai2(){
    export trace_type=akamai2
    export server_cache_size=1000
    export ats_ram_size=209715200
    export fe_ram_size=8589934592
    # export unavail_trace=config/unavailability.one
    export trace_path="s3://juncheng-data/C2DN/akamai2.bin.scale10"
}


testbed=unknown
system=unknown
workload=unknown
exp=unknown
unavailability=0
unavail_trace=""


while getopts ":t:s:w:e:u" arg; do
    case "${arg}" in
        t)
            testbed=${OPTARG}
            [ "$testbed" == "aws" ] || [ "$testbed" == "cloudlab" ] || usage
            ;;
        s)
            system=${OPTARG}
            [ "$system" == "CDN" ] || [ "$system" == "C2DN" ] || [ "$system" == "naiveCoding" ]  || [ "$system" == "noRep" ]|| usage
            ;;
        w)
            workload=${OPTARG}
            [ "$workload" == "akamai1" ] || [ "$workload" == "akamai2" ] || [ "$workload" == "akamai1Unavailability" ] || [ "$workload" == "akamai2Unavailability" ] || [ "$workload" == "macrobenchmark" ] || [ "$workload" == "akamai2" ] || usage
            ;;
        e)
            exp=${OPTARG}
            [[ "$exp" == "expLatency" ]] || [[ "$exp" == "expThrpt" ]] || [[ "$exp" == "expUnavailability" ]] || [[ "$exp" == "test"* ]] || [[ "$exp" == "monitor"* ]] || usage 
            ;;
        u) 
            unavailability=1
            unavail_trace=${OPTARG:-config/unavailability.one}
            ;;
        *)
            usage
            ;;
    esac
done
shift $((OPTIND-1))


[ "$testbed" == "unknown" ]     && echo "unknown testbed"   && usage && exit; 
[ "$system" == "unknown" ]      && echo "unknown system"    && usage && exit; 
[ "$workload" == "unknown" ]    && echo "unknown workload"  && usage && exit; 
[ "$exp" == "unknown" ]         && echo "unknown exp"       && usage && exit; 


[ "$testbed" == "aws" ]     && export user=ubuntu; 


export expname_BASE="${testbed}_${system}_${workload}_${exp}_unavail${unavailability}"

[ "$workload" == "akamai1" ] && akamai1 
[ "$workload" == "akamai2" ] && akamai2
[ "$workload" == "macrobenchmark" ] && macrobenchmark

[ "$system" == "CDN" ] && CDN 
[ "$system" == "C2DN" ] && C2DN 
[ "$system" == "noRep" ] && noRep 
[ "$system" == "naiveCoding" ] && naiveCoding

source ../scriptcore/exps.sh

[ "$exp" == "expLatency" ] && expLatency
[ "$exp" == "expThrpt" ] && expThrpt 
[ "$exp" == "expUnavailability" ] && expUnavailability
[ "$exp" == "monitorFEMetrics" ] && monitorFEMetrics
[ "$exp" == "test" ] && stopExp && collectResult || true
[ "$exp" == "test2" ] && runLocalClient 1 || true










