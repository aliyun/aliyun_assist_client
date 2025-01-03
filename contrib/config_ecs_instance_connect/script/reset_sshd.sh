#!/bin/bash

modified=false

# Remove ECS Instance Connect sshd config if present
AUTH_KEYS_CMD="#AuthorizedKeysCommand none"
AUTH_KEYS_USER="#AuthorizedKeysCommandUser nobody"

# Remove ECS Instance Connect sshd config if present
if grep -q "^AuthorizedKeysCommandUser[[:blank:]]ecs-instance-connect$" /etc/ssh/sshd_config || grep -q "^AuthorizedKeysCommandRunAs[[:blank:]]ecs-instance-connect$" /etc/ssh/sshd_config; then
    if grep -q "^AuthorizedKeysCommand[[:blank:]]/usr/bin/timeout[[:blank:]]5s[[:blank:]]/usr/local/share/aliyun-assist/plugin/config_ecs_instance_connect/ecs_config_instance_connect[[:blank:]]%u[[:blank:]]%f$" /etc/ssh/sshd_config ; then
        sed -i "/^.*AuthorizedKeysCommand[[:blank:]]\/usr\/bin\/timeout[[:blank:]]5s[[:blank:]]\/usr\/local\/share\/aliyun-assist\/plugin\/config_ecs_instance_connect\/ecs_config_instance_connect[[:blank:]]%u[[:blank:]]%f$/d" /etc/ssh/sshd_config
        sed -i "/^.*AuthorizedKeysCommandUser[[:blank:]]ecs-instance-connect$/d" /etc/ssh/sshd_config
        sed -i "/^.*AuthorizedKeysCommandRunAs[[:blank:]]ecs-instance-connect$/d" /etc/ssh/sshd_config
        modified=true
    fi
fi

if [ $modified = true ] ; then
    systemctl --version > /dev/null 2>&1
    if [ $? -eq '0' ];then
        sudo systemctl daemon-reload
        sudo systemctl list-units ssh.service | grep ssh.service -q  && sudo systemctl restart ssh.service && exit $?
        sudo systemctl list-units sshd.service | grep sshd.service -q  && sudo systemctl restart sshd.service && exit $?
        exit $?
    fi
    service --help > /dev/null 2>&1
    if [ $? -eq '0' ];then
        sudo service ssh restart || sudo service sshd restart
        exit $?
    fi
    sudo /etc/init.d/sshd restart
    exit $?
fi

/usr/bin/id -u "ecs-instance-connect" > /dev/null 2>&1
if [ $? -eq 0 ] ; then
    # Delete system user
    /usr/sbin/userdel ecs-instance-connect
fi
