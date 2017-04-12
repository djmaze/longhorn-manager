#!/bin/bash

ORC_TEST_PREFIX=longhorn-manager-test

ETCD_SERVER=${ORC_TEST_PREFIX}-etcd-server
ETCD_IMAGE=quay.io/coreos/etcd:v3.1.5

NFS_SERVER=${ORC_TEST_PREFIX}-nfs-server
NFS_IMAGE=docker.io/erezhorev/dockerized_nfs_server

LONGHORN_IMAGE=rancher/longhorn:7a918a0

BACKUPSTORE_PATH=/opt/backupstore

function check_exists {
    name=$1
    if [ "$(docker ps -aq -f status=exited -f name=${name})" ];
    then
	docker rm -v ${name}
    fi
    if [ "$(docker ps -q -f name=${name})" ]; then
        echo true
        return
    fi
    echo false
}

function start_etcd {
    exists=$(check_exists $ETCD_SERVER)

    if [ "$exists" == "true" ]; then
        echo etcd server exists
        return
    fi

    echo Start etcd server
    docker run -d \
                --name $ETCD_SERVER \
                --volume /etcd-data \
                $ETCD_IMAGE \
                /usr/local/bin/etcd \
                --name longhorn-test-etcd-1 \
                --data-dir /etcd-data \
                --listen-client-urls http://0.0.0.0:2379 \
                --advertise-client-urls http://0.0.0.0:2379 \
                --listen-peer-urls http://0.0.0.0:2380 \
                --initial-advertise-peer-urls http://0.0.0.0:2380 \
                --initial-cluster longhorn-test-etcd-1=http://0.0.0.0:2380 \
                --initial-cluster-token my-etcd-token \
                --initial-cluster-state new \
                --auto-compaction-retention 1

    echo etcd server is up
}

function cleanup_orc_test {
    echo clean up test containers
    set +e
    docker stop $(docker ps -f name=$ORC_TEST_PREFIX -a -q)
    docker rm -v $(docker ps -f name=$ORC_TEST_PREFIX -a -q)
    set -e
}

function wait_for {
    url=$1

    ready=false

    set +e
    for i in `seq 1 5`
    do
            sleep 1
            curl -sL --max-time 1 --fail --output /dev/null --silent $url
            if [ $? -eq 0 ]
            then
                    ready=true
                    break
            fi
    done
    set -e

    if [ "$ready" != true ]
    then
            echo Fail to wait for $url
            return -1
    fi
    return 0
}

function wait_for_etcd {
    etcd_ip=$1
    wait_for http://${etcd_ip}:2379/v2/stats/leader
}

function get_container_ip {
    container=$1
    for i in `seq 1 5`
    do
        ip=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $container`
        if [ "$ip" != "" ]
        then
            break
        fi
        sleep 10
    done

    if [ "$ip" == "" ]
    then
        echo cannot find ip for $container
        exit -1
    fi
    echo $ip
}

function start_longhorn_binary {
    name=${ORC_TEST_PREFIX}-longhorn-binary
    exists=$(check_exists $name)

    if [ "$exists" == "true" ]; then
        echo longhorn binary exists
        return
    fi

    image=$1
    docker run -d -it --name $name $image bash
    echo longhorn binary is up
}

function start_orc {
    image=$1
    id=$2
    etcd_ip=$3
    shift 3
    extra=$@

    name=${ORC_TEST_PREFIX}-server-${id}
    exists=$(check_exists $name)
    if [ "$exists" == "true" ]; then
        echo remove old longhorn-manager server
	docker stop ${name}
	docker rm -v ${name}
    fi

    docker run -d --name ${name} \
            --privileged -v /dev:/host/dev \
            -v /var/run:/var/run ${extra} \
            --volumes-from ${ORC_TEST_PREFIX}-longhorn-binary ${image} \
            /usr/local/sbin/launch-orc -d --orchestrator docker \
            --longhorn-image $LONGHORN_IMAGE \
            --etcd-servers http://${etcd_ip}:2379
    echo ${name} is up
}

function wait_for_orc {
    orc_ip=$1
    wait_for http://${orc_ip}:9500/v1
}

function start_nfs {
    name=${NFS_SERVER}
    exists=$(check_exists $name)

    if [ "$exists" == "true" ]; then
        echo nfs server exists
        return
    fi

    docker run -d --name ${NFS_SERVER} --privileged ${NFS_IMAGE} ${BACKUPSTORE_PATH}
    echo nfs server is up
}
