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

  Timed means period task，using --Frequency "0 */10 * * * *" set per 10 minutes run once。

cronat expression:
*       *      *    *   *      *
second minute hour day month week

0 */10 * * * *  every 10 minutes run 
0 30 21 * * * every 21:30 run
0 10 1 * * 6,0 run at 1:10 every Saturday and Sunday
0 0,30 18-23 * * * run every 30 minutes between 18:00 and 23:00 every day

c)Watch result：
  aliyuncli ecs DescribeInvocationResults --InstanceId your-vm-instance-id --InvokeId your-task-id
DescribeInvocations can watch the task status：
  aliyuncli ecs DescribeInvocations --InstanceId your-vm-instance-id --InvokeId your-task-id


  openapi：

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

using： 
  # ZWNobyAxMjM= is echo 123 base64 decode.except result is MTIzCg==(123)
  shell_command_id = create_command('ZWNobyAxMjM=', 'RunShellScript', 'test', 'test')
  invoke_id = invoke_command(instance_id, shell_command_id, 'false')
  # MTIzCg== base64 decode is 123, if task run susccess
  check_task_result(instance_id, invoke_id, 'MTIzCg==')

### Contributing

    Welcome use Github pull requests to commit.

## License

    aliyun assist is licensed under the GPL V3 License.