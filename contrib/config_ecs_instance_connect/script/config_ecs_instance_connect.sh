#!/usr/bin/env bash

print_help(){
cat  <<EOF
Usage:
	--install     install ecs instance connect
	--uninstall   uninstall ecs instance connect
EOF
}

src_dir="$( cd "$( dirname "${0}" )" && pwd )"
dst_dir="/usr/local/share/aliyun-assist/plugin/config_ecs_instance_connect"

install(){
chmod +x $src_dir/reset_sshd.sh
chmod +x $src_dir/configure_sshd.sh

cp -f $src_dir/ecs_config_instance_connect $dst_dir/ecs_config_instance_connect
chmod +x $dst_dir/ecs_config_instance_connect

bash $src_dir/configure_sshd.sh
exit_code=$?
if [ $exit_code -eq 0 ];then
	echo "install finished ..."
else
	echo "install falied"
	exit $exit_code
fi
}

uninstall(){

bash $src_dir/reset_sshd.sh
exit_code=$?
if [ $exit_code -eq 0 ];then
	rm -rf "$dst_dir/ecs_config_instance_connect"
	echo "uninstall finished ..."
else
	echo "uninstall failed"
	exit $exit_code
fi
}

main(){
	args=$1
	case $args in
		--install)
		install
		;;
		--uninstall)
		uninstall
		;;
		*)
		print_help
		;;
	esac
}

main $1