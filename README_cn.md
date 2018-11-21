### 阿里云助手

阿里云助手是一款支持远程执行运维任务的云服务产品，如：您可以使用云助手对运行中的Windows实例执行bat/powershell运维脚本，对运行中的Linux实例执行Shell脚本。

### 几个概念：

-   命令(command)：需要在实例中执行的具体操作，如具体的shell脚本
-   任务(Invocation)：选中某些目标实例来执行某个命令，即创建了一个任务(Invocation)
-   定时任务(Timed Invocation)：在创建任务时，您可以指定任务的执行时序/周期，就是定时任务(Timed Invocation)；定时任务(Timed Invocation)的目的主要是周期性的执行某些维护操作

### 系统要求

-   windows Server 2008/2012/2016
-   Ubuntu
-   CentOS
-   Debian
-   RedHat
-   SUSE Linux Enterprise Server
-   OpenSUSE
-   Aliyun Linux
-   FreeBSD
-   CoreOS

### [安装方法](https://help.aliyun.com/document_detail/64921.html)


### 如何编译

#### Windows：  
    1) cmake .  
    2) 用vs打开sln文件编译  

#### Linux：  
    1) cmake .  
    2) make  


### [如何使用](https://help.aliyun.com/document_detail/64741.html)


### Contributing

    欢迎使用Github的pull requests机制给我们提交代码.  

## License

    阿里云助手 is licensed under the GPL V3 License.  