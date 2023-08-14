// +build freebsd

package process

const sudoersFile = "/usr/local/etc/sudoers.d/ecs-assist-user"
const createUserCommandFormater = "pw useradd %s -m -s /sbin/nologin"
