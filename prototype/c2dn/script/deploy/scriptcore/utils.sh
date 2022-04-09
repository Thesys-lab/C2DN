#!/bin/bash


# export IFS=$'\n'


# ssh user@host <<-'ENDSSH'
#     #commands to run on remote host
#     ssh user@host2 <<-'END2'
#         # Another bunch of commands on another host
#         wall <<-'ENDWALL'
#             Error: Out of cheese
#         ENDWALL
#         ftp ftp.secureftp-test.com <<-'ENDFTP'
#             test
#             test
#             ls
#         ENDFTP
#     END2
# ENDSSH


info() {
    logging "INFO" "$1"
}

debug() {
    logging "DEBUG" "$1"
}

verbose1() {
    logging "VERB1" "$1"
}

verbose2() {
    logging "VERB2" "$1"
}

warning() {
    logging "WARN" "$1"
}

error() {
    logging "ERROR" "$1"
    exit 1
}

logging() {
    echo "$(date +%H:%M:%S) [$1]:  $2" 
}

slack() {
    curl -X POST -H 'Content-type: application/json' --data '{"text":"'${expname}' '${1:-}'"}' ${slackhook} 
}


function join_by { local IFS=" "; echo "$*"; shift; }
function join_by_comma { local IFS=","; echo "$*"; shift; }



askContinue(){
    [[ "${NO_CHECK}" == 1 ]] && return 

    tput bel  # ring bell
    [[ $# -ge 1 ]] && info "$1"
    info "Please check the information above, continue? "
    read y
    if [[ ! "$y" == "y"* ]]; then
        exit
    fi
}







