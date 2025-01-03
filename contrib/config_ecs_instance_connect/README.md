## 插件名称
  config_ecs_instance_connect
## 插件简介
  负责从云助手服务端获取临时秘钥以实现ssh临时秘钥登录功能。
## 实现原理
  安装此插件后会自动在`/etc/ssh/sshd_config`中增加以下配置内容，再通过ssh秘钥登录时`ecs_config_instance_connect`将被执行，它将云助手服务端拉取暂存在服务端的临时秘钥。
  ```shell
AuthorizedKeysCommand /usr/bin/timeout 5s /usr/local/share/aliyun-assist/plugin/config_ecs_instance_connect/ecs_config_instance_connect %u %f
  ```
## 兼容性说明
  适用于Linux x64和arm64环境
## 打包方式
* for linux x64
```shell
arch=x64
dist_dir=dist_${arch}
rm -rf $dist_dir
mkdir $dist_dir
sed "s/{{ ARCH }}/${arch}/" config.json > $dist_dir/config.json
cp script/*.sh $dist_dir
GOOS=linux GOARCH=amd64 go build -o $dist_dir/ecs_config_instance_connect
cd $dist_dir
zip -r config_ecs_instance_connect_${arch}.zip ./*
cd -
```

* for linux arm
```shell
arch=arm
dist_dir=dist_${arch}
rm -rf $dist_dir
mkdir $dist_dir
sed "s/{{ ARCH }}/${arch}/" config.json > $dist_dir/config.json
cp script/*.sh $dist_dir
GOOS=linux GOARCH=amr64 go build -o $dist_dir/ecs_config_instance_connect
cd $dist_dir
zip -r config_ecs_instance_connect_${arch}.zip ./*
cd -
```

## 使用指南
```shell
# 安装
acs-plugin-manager -e -P config_ecs_instance_connect -p --install
# 卸载
acs-plugin-manager -e -P config_ecs_instance_connect -p --uninstall
```