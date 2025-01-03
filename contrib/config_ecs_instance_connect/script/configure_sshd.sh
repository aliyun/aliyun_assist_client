#!/bin/bash

# Create/configure system user
/usr/bin/id -u "ecs-instance-connect" > /dev/null 2>&1
if [ $? -ne 0 ] ; then
    /usr/bin/getent passwd ecs-instance-connect || /usr/sbin/useradd -r -M -s /sbin/nologin ecs-instance-connect
    /usr/sbin/usermod -L ecs-instance-connect
fi

modified=false

# Configure sshd to use ECS Instance Connect's AuthorizedKeysCommand
AUTH_KEYS_CMD="AuthorizedKeysCommand /usr/bin/timeout 5s /usr/local/share/aliyun-assist/plugin/config_ecs_instance_connect/ecs_config_instance_connect %u %f"
AUTH_KEYS_USR="AuthorizedKeysCommandUser ecs-instance-connect"
if ssh -v  2>&1 | grep "OpenSSH_5." -q; then
    AUTH_KEYS_USR="AuthorizedKeysCommandRunAs ecs-instance-connect"
fi
if ! grep -q "^[^#]*AuthorizedKeysCommand[[:blank:]]\+.*$" /etc/ssh/sshd_config ; then
    if ! grep -q "^[^#]*AuthorizedKeysCommandUser[[:blank:]]\+.*$" /etc/ssh/sshd_config ; then
        if ! grep -q "^[^#]*AuthorizedKeysCommandRunAs[[:blank:]]\+.*$" /etc/ssh/sshd_config ; then
            # Add our configuration
            printf "\n%s\n%s\n" "${AUTH_KEYS_CMD}" "${AUTH_KEYS_USR}" >> /etc/ssh/sshd_config
            modified=true
        fi
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
