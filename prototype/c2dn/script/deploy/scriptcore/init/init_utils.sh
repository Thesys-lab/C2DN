#!/bin/bash


trap 'previous_command=$this_command; this_command=$BASH_COMMAND' DEBUG
# trap 'echo error at line ${LINENO}' ERR
trap 'echo "exit status $? due to command \"$this_command\""' EXIT


# let script exit if a command fails
set -o errexit
#let script exit if an unsed variable is used
set -o nounset


logging_level=4

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
    level=0
    case "$1" in 
        "ERROR")
            level=2
            ;;
        "WARN")
            level=3
            ;;
        "INFO")
            level=4
            ;;
        "DEBUG")
            level=5
            ;;
        "VERB1")
            level=6
            ;;
        "VERB2")
            level=7
            ;;
    esac

    [ $level -le ${logging_level} ] && echo -e "$(date +%H:%M:%S) [$1]:    \t$(hostname): $2" || true
}




