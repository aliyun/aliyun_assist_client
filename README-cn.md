# 阿里云助手

阿里云助手是一种可以帮您自动执行各种运维任务的能力，如：您可以使用云助手对运行中的Windows实例执行bat/powershell运维脚本，对运行中的Linux实例执行Shell脚本。

几个概念：
命令(command)：需要在实例中执行的具体操作，如具体的shell脚本
任务(Invocation)：选中某些目标实例来执行某个命令，即创建了一个任务(Invocation)
定时任务(Timed Invocation)：在创建任务时，您可以指定任务的执行时序/周期，就是定时任务(Timed Invocation)；定时任务(Timed Invocation)的目的主要是周期性的执行某些维护操作

### 系统要求:

windows Server 2008/2012/2016
Ubuntu   
CentOS  
Debian
RedHat
SUSE Linux Enterprise Server
OpenSUSE
Aliyun Linux
FreeBSD
CoreOS

### 安装方法:
若您的实例中未安装云助手的客户端，请安装如下步骤进行安装：
Windows实例：
以管理员权限安装：
    http://repository-iso.oss-cn-beijing.aliyuncs.com/download/aliyun_agent_setup.exe

Linux实例：
rpm包地址：
    http://repository-iso.oss-cn-beijing.aliyuncs.com/download/aliyun_assist.rpm
Deb包地址：
    http://repository-iso.oss-cn-beijing.aliyuncs.com/download/aliyun_assist.deb
		
无独立IP，各个Region的下载方式:
  http://axt.{region_name}.alibaba-inc.com:8080/assist/aliyun_assist.deb
  http://axt.{region_name}.alibaba-inc.com:8080/assist/aliyun_assist.rpm
  http://axt.{region_name}.alibaba-inc.com:8080/assist/aliyun_assist.exe
如region是杭州，则下载地址对应于 http://axt.ch-hangzhou.alibaba-inc.com:8080/assist/aliyun_assist.deb

### 文件结构:

  /service  服务框架
../task_engine 云助手任务引擎
../package_installer 云助手软件安装
../test  单元测试
../third_party 第三方库
../common 云助手common库
../update 软件自升级
	
### 如何编译
    Windows：
		1) cmake .
		2) 用vs打开sln文件编译
		
    Linux：
		1) cmake .
		2) make
		

### 如何使用

  aliyuncli方式：
 
  首先安装aliyuncli和aliyun openapi sdk：
1 pip install aliyuncli
2 pip install aliyun-python-sdk-core
3 pip install aliyun-python-sdk-axt
	
由于我们修改了aliyuncli对于数组的支持，下载我们的aliyuncli的aliyunOpenApiData.py文件去替换%python_install_path%\Lib\site-packages\aliyuncli\aliyunOpenApiData.py
  下载地址：http://repository-iso.oss-cn-beijing.aliyuncs.com/cli/aliyunOpenApiData.py
  Linux参考：
  linux(ubuntu)
    /usr/local/lib/python2.7/dist-packages   
  linux(redhat)
    /usr/lib/python2.7/site-packages
	
  配置用户key和region
$ aliyuncli configure
Aliyun Access Key ID [None]: <Your aliyun access key id>
Aliyun Access Key Secret [None]: <Your aliyun access key secret>
Default Region Id [None]: cn-hangzhou
Default output format [None]: 

a)创建命令：
  aliyuncli ecs CreateCommand --CommandContent ZWNobyAxMjM= --Type RunBatScript --Name test --Description test
其中 CommandContent中的ZWNobyAxMjM=为将'echo 123'经base64后转化的编码,如果目标实例的操作系统类型是linux，type改为RunShellScript。
创建成功后，将返回command-id信息


b)选中目标实例执行命令：
  aliyuncli ecs InvokeCommand --InstanceIds  your-vm-instance-id1 instance-id2 --CommandId your-command-id --Timed false
其中Timed表示是否周期性任务，通过设置--Frequency "0 */10 * * * *" 将该任务设置为每10分钟执行一次，其中上面的描述为cron表达式。
执行成功后将为所有的目标实例返回一个统一的额invokeid，后续可使用该invokeid查询命令的执行情况

c)查看执行结果：
  aliyuncli ecs DescribeInvocationResults --InstanceId your-vm-instance-id --InvokeId your-task-id
其中DescribeInvocations可以查看该任务的执行状态：
  aliyuncli ecs DescribeInvocations --InstanceId your-vm-instance-id --InvokeId your-task-id
	
  openapi方式：

### Contributing

    欢迎使用Github的pull requests机制给我们提交代码.

## License

    阿里云助手 is licensed under the GPL V3 License.