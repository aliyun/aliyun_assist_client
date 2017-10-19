# Aliyun Assist

Aliyun assist can help you automatically perform various tasks such as:
operation ability, you can use the aliyun assist executive bat/powershell operation script on a running instance of Windows, and Shell script on instance of Linux.

Concept
  Command：Specific operations that need to be executed in an instance, such as a specific shell script.
  Invocation：Select some target instances to execute a command.
  Timed Invocation：When you create a task, you can specify the execution sequence / cycle of the task, which is the Timed Invocation.

###  Verify Requirements:

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

### Setup:
If your vm has not install yet, please：
Windows：
run as administrator：
    http://repository-iso.oss-cn-beijing.aliyuncs.com/download/aliyun_agent_setup.exe

Linux：
rpm package：
    http://repository-iso.oss-cn-beijing.aliyuncs.com/download/aliyun_assist.rpm
Deb package：
    http://repository-iso.oss-cn-beijing.aliyuncs.com/download/aliyun_assist.deb
		
If you can not connect intetnet, then download:
  http://axt.{region_name}.alibaba-inc.com:8080/assist/aliyun_assist.deb
  http://axt.{region_name}.alibaba-inc.com:8080/assist/aliyun_assist.rpm
  http://axt.{region_name}.alibaba-inc.com:8080/assist/aliyun_assist.exe
For example, if region is hangzhou，then you can use http://axt.ch-hangzhou.alibaba-inc.com:8080/assist/aliyun_assist.deb

### File Structure:

  /service  service framework
../task_engine assist task engine
../package_installer software install
../test unit test
../third_party third party lib
../common assist common lib
../update assist update
	
### How to compile
    Windows：
		1) cmake .
		2) open .sln using vs
		
    Linux：
		1) cmake .
		2) make
		

### How to use
  aliyuncli：
 
  Install aliyuncli和aliyun openapi sdk：
1 pip install aliyuncli
2 pip install aliyun-python-sdk-core
3 pip install aliyun-python-sdk-axt
	
Because we modify origin aliyun，please download aliyunOpenApiData.py to replace %python_install_path%\Lib\site-packages\aliyuncli\aliyunOpenApiData.py
  Download url：http://repository-iso.oss-cn-beijing.aliyuncs.com/cli/aliyunOpenApiData.py
  Under Linux system：
  linux(ubuntu)
    /usr/local/lib/python2.7/dist-packages   
  linux(redhat)
    /usr/lib/python2.7/site-packages
	
  Config key and region
$ aliyuncli configure
Aliyun Access Key ID [None]: <Your aliyun access key id>
Aliyun Access Key Secret [None]: <Your aliyun access key secret>
Default Region Id [None]: cn-hangzhou
Default output format [None]: 

a)Create command：
  aliyuncli ecs CreateCommand --CommandContent ZWNobyAxMjM= --Type RunBatScript --Name test --Description test
In CommandContent, 'ZWNobyAxMjM=' is 'echo 123' base64 decoded string,if linux，type should be RunShellScript, return command-id


b)Invoke task：
  aliyuncli ecs InvokeCommand --InstanceIds  your-vm-instance-id1 instance-id2 --CommandId your-command-id --Timed false


c)Watch result：
  aliyuncli ecs DescribeInvocationResults --InstanceId your-vm-instance-id --InvokeId your-task-id
DescribeInvocations can watch the task status：
  aliyuncli ecs DescribeInvocations --InstanceId your-vm-instance-id --InvokeId your-task-id
 
### Contributing

    Welcome use Github pull requests to commit.

## License

    aliyun assist is licensed under the GPL V3 License.