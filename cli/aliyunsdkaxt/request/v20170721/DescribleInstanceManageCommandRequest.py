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
class DescribleInstanceManageCommandRequest(RpcRequest):

	def __init__(self):
		RpcRequest.__init__(self, 'axt', '2017-07-21', 'DescribleInstanceManageCommand')

	def get_pageSize(self):
		return self.get_query_params().get('pageSize')

	def set_pageSize(self,pageSize):
		self.add_query_param('pageSize',pageSize)

	def get_pageNumber(self):
		return self.get_query_params().get('pageNumber')

	def set_pageNumber(self,pageNumber):
		self.add_query_param('pageNumber',pageNumber)

	def get_description(self):
		return self.get_query_params().get('description')

	def set_description(self,description):
		self.add_query_param('description',description)

	def get_name(self):
		return self.get_query_params().get('name')

	def set_name(self,name):
		self.add_query_param('name',name)

	def get_InstanceManageCommandId(self):
		return self.get_query_params().get('InstanceManageCommandId')

	def set_InstanceManageCommandId(self,InstanceManageCommandId):
		self.add_query_param('InstanceManageCommandId',InstanceManageCommandId)

	def get_type(self):
		return self.get_query_params().get('type')

	def set_type(self,type):
		self.add_query_param('type',type)