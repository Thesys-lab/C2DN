#!/bin/bash 


export SEPARATOR='######################################################'

# if WORKING_DIR is empty, then set to curr dir and make readonly 
[[ "${WORKING_DIR:-}" == "" ]] && readonly WORKING_DIR=$(pwd) || true
# find the absolute path of this script 
[[ "${scriptcore_dir:-}" == "" ]] && readonly scriptcore_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )" || true
[[ "${launch_script_dir:-}" == "" ]] && readonly launch_script_dir=$(realpath ${scriptcore_dir}/../launchscript/) || true
[[ "${top_level_dir:-}" == "" ]] && readonly top_level_dir=$(realpath ${scriptcore_dir}/../../../) || true
[[ "${remote_exp_dir:-}" == "" ]] && readonly remote_exp_dir=CDN/ || true


source "${scriptcore_dir}/assert.sh"
assert_not_eq "${scriptcore_dir}" "" "cannot find base dir"
assert_eq "${scriptcore_dir##*/}" "scriptcore" "base dir name changes"



