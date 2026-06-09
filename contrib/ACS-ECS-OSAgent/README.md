## 插件名称
  ACS-ECS-OSAgent

## 插件简介
  专业执行 OS 管理和运维等更多任务的 AI Agent。

## 实现原理
  基于 Node.js 运行时、TypeScript 编程语言、React+Ink 界面库开发的 AI Agent，通过官方 SDK 支持 MCP、ACP 等扩展协议。

## 兼容性说明
  截至目前支持以下操作系统和架构：
  * Linux x64 平台
  * Linux arm64 平台
  * Windows x64 平台

## 打包方式
  `axt-plugin/` 目录中提供了用于构建插件包的 `Makefile` 构建脚本，以及用于制作不同平台插件包的插件描述文件模板 `config.json.in`、执行入口文件 `entrypoint.sh` 和 `entrypoint.bat`、以及其它文件。执行以下命令即可构建 AI Agent 本身并打包 Node.js 运行时和依赖库成云助手插件包。

  ```shell
  make -C axt-plugin/
  ```

## 使用指南
  在使用插件之前，需要先编辑配置文件 `<用户主目录>/.osagent/config.yaml` 配置可用的大模型服务 API。配置文件格式如下：

  ```yaml
  models:
    <模型配置标识>:                          # 支持[a-zA-Z0-9/._-]字符，但必须以[a-zA-Z0-9]开头
      type: openai-compatible                # 当前只支持 OpenAI 兼容格式
      api_key: "<API Key>"                   # 必填
      base_url: "<OpenAI兼容API的起始地址>"  # 例如：https://dashscope.aliyuncs.com/compatible-mode/v1
      model: "<模型服务支持的模型id>"        # 例如：qwen3.6-plus

  defaultModel: <默认要使用的模型配置标识>   # 必填
  ```

  然后执行以下命令即可启动插件。即使没有按照上面的步骤预先进行配置，插件启动时也会检测到缺少可用的大模型服务而主动提示。

  ```shell
  acs-plugin-manager --exec --plugin ACS-ECS-OSAgent
  ```

  可以通过以下命令查看插件的帮助信息了解更多用法：

  ```shell
  acs-plugin-manager --exec --plugin ACS-ECS-OSAgent --params "--help"
  ```
