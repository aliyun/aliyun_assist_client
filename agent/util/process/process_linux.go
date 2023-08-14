// +build linux

package process

const sudoersFile = "/etc/sudoers.d/ecs-assist-user"
const createUserCommandFormater = "useradd -m %s -s /sbin/nologin"
