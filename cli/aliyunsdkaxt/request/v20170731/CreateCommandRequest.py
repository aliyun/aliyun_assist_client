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
class CreateCommandRequest(RpcRequest):

	def __init__(self):
		RpcRequest.__init__(self, 'axt', '2017-07-31', 'CreateCommand')

	def get_WorkingDir(self):
		return self.get_query_params().get('WorkingDir')

	def set_WorkingDir(self,WorkingDir):
		self.add_query_param('WorkingDir',WorkingDir)

	def get_Description(self):
		return self.get_query_params().get('Description')

	def set_Description(self,Description):
		self.add_query_param('Description',Description)

	def get_CommandContent(self):
		return self.get_query_params().get('CommandContent')

	def set_CommandContent(self,CommandContent):
		self.add_query_param('CommandContent',CommandContent)

	def get_Name(self):
		return self.get_query_params().get('Name')

	def set_Name(self,Name):
		self.add_query_param('Name',Name)

	def get_Type(self):
		return self.get_query_params().get('Type')

	def set_Type(self,Type):
		self.add_query_param('Type',Type)

	def get_TimeOut(self):
		return self.get_query_params().get('TimeOut')

	def set_TimeOut(self,TimeOut):
		self.add_query_param('TimeOut',TimeOut)