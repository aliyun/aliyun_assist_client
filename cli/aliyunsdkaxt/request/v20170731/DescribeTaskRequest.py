# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
#
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

from aliyunsdkcore.request import RpcRequest
class DescribeTaskRequest(RpcRequest):

	def __init__(self):
		RpcRequest.__init__(self, 'axt', '2017-07-31', 'DescribeTask')

	def get_PageSize(self):
		return self.get_query_params().get('PageSize')

	def set_PageSize(self,PageSize):
		self.add_query_param('PageSize',PageSize)

	def get_Timed(self):
		return self.get_query_params().get('Timed')

	def set_Timed(self,Timed):
		self.add_query_param('Timed',Timed)

	def get_CommandId(self):
		return self.get_query_params().get('CommandId')

	def set_CommandId(self,CommandId):
		self.add_query_param('CommandId',CommandId)

	def get_PageNumber(self):
		return self.get_query_params().get('PageNumber')

	def set_PageNumber(self,PageNumber):
		self.add_query_param('PageNumber',PageNumber)

	def get_TaskId(self):
		return self.get_query_params().get('TaskId')

	def set_TaskId(self,TaskId):
		self.add_query_param('TaskId',TaskId)

	def get_ItemStatus(self):
		return self.get_query_params().get('ItemStatus')

	def set_ItemStatus(self,ItemStatus):
		self.add_query_param('ItemStatus',ItemStatus)

	def get_CommandType(self):
		return self.get_query_params().get('CommandType')

	def set_CommandType(self,CommandType):
		self.add_query_param('CommandType',CommandType)

	def get_TaskStatus(self):
		return self.get_query_params().get('TaskStatus')

	def set_TaskStatus(self,TaskStatus):
		self.add_query_param('TaskStatus',TaskStatus)

	def get_CommandName(self):
		return self.get_query_params().get('CommandName')

	def set_CommandName(self,CommandName):
		self.add_query_param('CommandName',CommandName)

	def get_InstanceId(self):
		return self.get_query_params().get('InstanceId')

	def set_InstanceId(self,InstanceId):
		self.add_query_param('InstanceId',InstanceId)