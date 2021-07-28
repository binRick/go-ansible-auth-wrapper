#!/bin/bash
set -e
cd $( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
ARGS="$@"
cmd="go get && make build && ./bin/ansible-auth-wrapper $ARGS"

nodemon -w . -e go,sh,mod,sum -x sh -- -c "($cmd; echo $?)||true"
