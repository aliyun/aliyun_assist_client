阿里云助手

阿里云助手是一种可以帮您自动执行各种运维任务的能力，如：您可以使用云助手对运行中的Windows实例执行bat/powershell运维脚本，对运行中的Linux实例执行Shell脚本。

### 几个概念：

-   命令(command)：需要在实例中执行的具体操作，如具体的shell脚本
-   任务(Invocation)：选中某些目标实例来执行某个命令，即创建了一个任务(Invocation)
-   定时任务(Timed Invocation)：在创建任务时，您可以指定任务的执行时序/周期，就是定时任务(Timed Invocation)；定时任务(Timed Invocation)的目的主要是周期性的执行某些维护操作

### 系统要求:

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
  http://axt-repo.{region_name}.alibaba-inc.com:8080/assist/aliyun_assist.deb
  http://axt-repo.{region_name}.alibaba-inc.com:8080/assist/aliyun_assist.rpm
  http://axt-repo.{region_name}.alibaba-inc.com:8080/assist/aliyun_assist.exe
如region是杭州，则下载地址对应于 http://axt-repo.ch-hangzhou.alibaba-inc.com:8080/assist/aliyun_assist.deb

### 文件结构:

../service  服务框架
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
-   1 pip install aliyuncli
-   2 pip install aliyun-python-sdk-core
-   3 pip install aliyun-python-sdk-axt
	
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
Type包括：
-   RunBatScript
-   RunShellScript
-   RunPowerShellScript

b)选中目标实例执行命令：
  aliyuncli ecs InvokeCommand --InstanceIds  your-vm-instance-id1 instance-id2 --CommandId your-command-id --Timed false
其中Timed表示是否周期性任务，通过设置--Frequency "0 */10 * * * *" 将该任务设置为每10分钟执行一次，其中上面的描述为cron表达式。
执行成功后将为所有的目标实例返回一个统一的额invokeid，后续可使用该invokeid查询命令的执行情况

cronat表达式使用:
*       *      *    *   *      *
second minute hour day month week

0 */10 * * * *  每隔10分钟执行一次
0 30 21 * * * 每晚的21:30执行一次
0 10 1 * * 6,0 每周六、周日的1 : 10执行
0 0,30 18-23 * * * 在每天18 : 00至23 : 00之间每隔30分钟执行

一些脚本示例：

将本地80端口的请求转发到8080端口，当前主机IP为192.168.1.80
iptables -t nat -A PREROUTING -d 192.168.1.80 -p tcp --dport 80 -j DNAT --to-destination 192.168.1.80:8080

删除5天前的文件
find /data -mtime +5 -type f -exec rm -rf{} \;

c)查看执行结果：
  aliyuncli ecs DescribeInvocationResults --InstanceId your-vm-instance-id --InvokeId your-task-id
其中DescribeInvocations可以查看该任务的执行状态：
  aliyuncli ecs DescribeInvocations --InstanceId your-vm-instance-id --InvokeId your-task-id

openapi方式：

from aliyunsdkecs.request.v20140526.CreateCommandRequest import CreateCommandRequest
from aliyunsdkecs.request.v20140526.InvokeCommandRequest import InvokeCommandRequest
from aliyunsdkecs.request.v20140526.DescribeInvocationResultsRequest import DescribeInvocationResultsRequest

def create_command(command_content, type, name, description):
    request = CreateCommandRequest()
    request.set_CommandContent(command_content)
    request.set_Type(type)
    request.set_Name(name)
    request.set_Description(description)
    response = _send_request(request)
    command_id = response.get('CommandId')
    return command_id;

def invoke_command(instance_id, command_id, timed):
    request = InvokeCommandRequest()
    request.set_Timed(timed)
    InstanceIds = [instance_id]
    request.set_InstanceIds(InstanceIds)
    request.set_CommandId(command_id)
    response = _send_request(request)
    invoke_id = response.get('InvokeId')
    return invoke_id;

def check_task_result(instance_id, invoke_id, result):
    detail = get_task_detail_by_id(instance_id, invoke_id, result)
    index = 0
    while detail is None and index < 30:
        detail = get_task_detail_by_id(instance_id, invoke_id, result)
        time.sleep(1)
        index+=1
    if detail is None:
        return 'false'
    else:
        return 'true';

def get_task_detail_by_id(instance_id, invoke_id, result):
    logging.info("Check instance %s invoke_id is %s", instance_id, invoke_id)
    request = DescribeInvocationResultsRequest()
    request.set_InstanceId(instance_id)
    request.set_InvokeId(invoke_id)
    response = _send_request(request)
    invoke_detail = None
    if response is not None:
        result_list = response.get('Invocation').get('ResultLists').get('ResultItem')
        for item in result_list:
            if item.get('Output') == result:
                invoke_detail = item
                break;
        return invoke_detail;

main函数: 
  # ZWNobyAxMjM= is echo 123 base64 decode.except result is MTIzCg==(123)
  shell_command_id = create_command('ZWNobyAxMjM=', 'RunShellScript', 'test', 'test')
  invoke_id = invoke_command(instance_id, shell_command_id, 'false')
  # MTIzCg== base64 decode is 123, if task run susccess
  check_task_result(instance_id, invoke_id, 'MTIzCg==')

### Contributing

    欢迎使用Github的pull requests机制给我们提交代码.

## License

    阿里云助手 is licensed under the GPL V3 License.