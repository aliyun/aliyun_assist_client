#!/bin/bash

assist_version='1.0.0.0'
current_path=`dirname $0`

mkdir -p /usr/local/share/aliyun-assist/$assist_version
cp -f $current_path/output/aliyun-service /usr/local/share/aliyun-assist/$assist_version/aliyun-service
cp -f $current_path/output/aliyun_assist_update /usr/local/share/aliyun-assist/$assist_version/aliyun_assist_update
cp -f $current_path/output/aliyun_installer /usr/local/share/aliyun-assist/$assist_version/aliyun_installer

chmod a+x /usr/local/share/aliyun-assist/$assist_version/aliyun-service
chmod a+x /usr/local/share/aliyun-assist/$assist_version/aliyun_assist_update
chmod a+x /usr/local/share/aliyun-assist/$assist_version/aliyun_installer

des_dir=""
script_dir="$current_path/init"
src_dir=$script_dir
echo "script_dir:$script_dir"
echo "assist_version:$assist_version"

issue=""
. "$script_dir/script/common/identify_issue_version"
echo "issue:$issue"


if [ ! -e $script_dir/script/$issue/common/update_service ]; then
    echo "3131:failed to find update script."
    #exit 1
else
    ######### compare the pv in vm and newest pv##########
    . "$script_dir/script/$issue/common/update_service"
    if [ $? -ne 0 ]; then
        echo "3131:failed to find update gshelld script."
        #exit 1
    fi
fi

/etc/init.d/agentwatch reload
#echo "0:success"
