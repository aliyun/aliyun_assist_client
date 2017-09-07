CMAKE_MINIMUM_REQUIRED(VERSION 3.1)
# CMake 最低版本号要求
PROJECT (assist) 

set(CMAKE_MODULE_PATH ${CMAKE_MODULE_PATH} "${CMAKE_SOURCE_DIR}/third_party/CMakeHelpers")

include(generate_product_version)

if(DEFINED ENV{ASSIST_VERSION_MAJOR})
message("use defined version")

generate_product_version(ProductVersionFiles
    NAME aliyun_agent
    VERSION_MAJOR $ENV{ASSIST_VERSION_MAJOR}
    VERSION_MINOR $ENV{ASSIST_VERSION_MINOR}
    VERSION_PATCH $ENV{ASSIST_VERSION_PATCH}
    VERSION_REVISION $ENV{ASSIST_VERSION_REVISION}
    COMPANY_NAME Alibaba)
	
else()
message("use default version")

generate_product_version(ProductVersionFiles
    NAME aliyun_agent
    VERSION_MAJOR 1
    VERSION_MINOR 0
    VERSION_PATCH 0
    VERSION_REVISION 99
    COMPANY_NAME Alibaba)
endif()


if(UNIX)
	SET(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -m32")
	SET(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -m32")
endif()

file(INSTALL ${CMAKE_CURRENT_SOURCE_DIR}/config/host.conf DESTINATION ${CMAKE_CURRENT_SOURCE_DIR}/output/config)

#if(UNIX)
#file(COPY ${CMAKE_CURRENT_SOURCE_DIR}/init DESTINATION ${CMAKE_CURRENT_SOURCE_DIR}/output)
#endif()

if(WIN32)
file(INSTALL ${CMAKE_CURRENT_SOURCE_DIR}/tools/build/install.bat DESTINATION ${CMAKE_CURRENT_SOURCE_DIR}/output)
endif()

add_subdirectory(third_party)
add_subdirectory(common)
add_subdirectory(update)
add_subdirectory(package_installer)
add_subdirectory(service)









