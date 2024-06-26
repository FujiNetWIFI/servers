#!/usr/bin/env bats -T  --print-output-on-failure

# https://github.com/bats-core

#     echo "status = ${status}"    >&3
#     echo "output = ${output}"    >&3
#     echo "lines[] = ${lines[@]}" >&3

#BATS_TEST_TIMEOUT=15

# extract the return code from any json
function getsuccess()
{
    echo $1 | jq -r ".success"
}

# extract a value from any json
function get() 
{
    PARAM="$1"
    JSON="$2"

    echo $2 | jq -r ".$PARAM"
}

function genprivkey() {
    LC_ALL=C tr -dc '[:graph:]' < /dev/urandom | fold -w 16 | head -n1
}

function genrandomtoken() {
# this must return: 0–9, A–Z, a–z, !#$%&()*+-;<=>?@^_`{|}~ to be ascii85 RFC1924

    ASCII85RFC='0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!#$%&()*+-;<=>?@^_`{|}~'
  #  LC_ALL=C tr -dc '[:graph:]' < /dev/urandom | fold -w 100 | head -n1 | tr -Cd "$FILTEROUT" TOBEDELETED
     LC_ALL=C tr -dc $ASCII85RFC < /dev/urandom | fold -w 40 | head -n1 
}

@test "curl installed" {
    command -v curl  &> /dev/null
}

@test "pgrep installed" {
    command -v pgrep  &> /dev/null
}

@test "FujinetID is running" {
    pgrep fujinet-id  &> /dev/null
}

@test "request a new pubkey" {
    privkey=$(genprivkey)
    
    run curl -k -s -X POST -H "Content-Type: application/json" https://localhost/genPubKey -d '{"privkey": "'"rogersm#${privkey}"'"}'
    [ $(getsuccess "${output}") = true ]
    token=$(get "token" "${output}" )
    echo "${token}" > test_token.id
    pubkey=$(get "pubkey" "${output}" )
    echo "${pubkey}" > test_pubkey.id
}

@test "retrieve pubkey from token" {
    token=$(cat test_token.id)
    pubkey=$(cat test_pubkey.id)

    run curl -k -s -X POST -H "Content-Type: application/json" https://localhost/getPubKey -d '{"token": "'"${token}"'"}'
    [ $(getsuccess  "${output}" ) = true ]
    curpubkey=$(get "pubkey" "${output}" )
    [ "${pubkey}" = "${curpubkey}" ]
    rm test_pubkey.id test_token.id
}

@test "fail retrieving an invalid token" {
    token=$(genrandomtoken)

    run curl -k -s -X POST -H "Content-Type: application/json" https://localhost/getPubKey -d '{"token": "'"${token}"'"}'
    [ $(getsuccess  "${output}") = false ]
}