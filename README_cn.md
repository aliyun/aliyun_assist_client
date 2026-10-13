### 阿里云助手

阿里云助手是一款支持远程执行运维任务的云服务产品，如：您可以使用云助手对运行中的Windows实例执行bat/powershell运维脚本，对运行中的Linux实例执行Shell脚本。

### 几个概念：

-   命令(command)：需要在实例中执行的具体操作，如具体的shell脚本
-   任务(Invocation)：选中某些目标实例来执行某个命令，即创建了一个任务(Invocation)
-   定时任务(Timed Invocation)：在创建任务时，您可以指定任务的执行时序/周期，就是定时任务(Timed Invocation)；定时任务(Timed Invocation)的目的主要是周期性的执行某些维护操作

### 系统要求

-   Windows Server 2012+
-   Alibaba Cloud Linux
-   AlmaLinux 8+
-   Anolis OS 7+
-   CentOS 6+
-   Debian 8+
-   Ubuntu 12+
-   RHEL 6+
-   SUSE 11 SP1+
-   Fedora 33+
-   CoreOS
-   OpenSUSE
-   FreeBSD 12+

### [安装方法](https://help.aliyun.com/zh/ecs/user-guide/install-the-cloud-assistant-agent)



### [如何使用](https://help.aliyun.com/zh/ecs/user-guide/java-programs-use-cloud-assistant-to-manage-ecs-instances)

### 插件

本仓库在 [contrib/](contrib/) 目录下开发了部分云助手插件：

- [ACS-ECS-OSAgent](contrib/ACS-ECS-OSAgent/)：专业执行 OS 管理和运维等更多任务的 AI Agent
- [config_ecs_instance_connect](contrib/config_ecs_instance_connect/)：从云助手服务端获取临时秘钥，实现 SSH 临时秘钥登录功能

### Contributing

    欢迎使用Github的pull requests机制给我们提交代码.  
