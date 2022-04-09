#!/bin/bash 
# DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"


export warmup_concurrency=8
export warmup_speedup=5
export uniq_obj_warmup=1
export use_backuped_disk_cache=0
export use_remote_origin=1


export NO_CHECK=1
export FORCE_INIT=1


# export cdn_host_file=/tmp/cdnPub

# param on CDN 
export OUTPUT_DIR="/tmp/c2dn/output/"
export LOCAL_WORKING_DIR='$HOME/CDN/c2dn/'

# export SUPPORT_BUCKET=yes
# export MACRO_BENCHMARK=0

# export DEBUG_MODE=1

# export ${remote_exp_dir}
# export ${BASE_DIR}
# export ${top_level_dir}


export origin_port=2048
export ats_port=8080
export fe_port=8081


export warmup_disk_start_ts=0
export warmup_disk_end_ts=36000
# export warmup_disk_end_ts=3600
export warmup_ram_start_ts=$(echo ${warmup_disk_end_ts}/10*9 | bc) 
export warmup_ram_end_ts=${warmup_disk_end_ts}
export eval_start_ts=${warmup_ram_end_ts}
# export eval_end_ts=48000
export eval_end_ts=64000
# export eval_end_ts=84000


