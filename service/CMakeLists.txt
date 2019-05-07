# CMake 最低版本要求
# CMake_minimum_required (VERSION 2.8)

# 项目信息
project (aliyun-assist)

include_directories(${CMAKE_CURRENT_BINARY_DIR})
include_directories(../third_party)
include_directories(../common)
include_directories(../task_engine)

aux_source_directory(../task_engine/plugin DIR_SRCS)
aux_source_directory(../task_engine DIR_SRCS)
aux_source_directory(../task_engine/notifer DIR_SRCS)
aux_source_directory(./ DIR_SRCS)

link_libraries(assist-common)
link_libraries(third_party)

if(WIN32)
    add_definitions(-DUNICODE -D_UNICODE)  
    include_directories(./windows)
    aux_source_directory(./windows DIR_SRCS)
    link_libraries(shlwapi Psapi version)
    link_libraries(ws2_32 Wldap32 libcurl libeay32 ssleay32 PowrProf)
    include_directories(../third_party/libcurl/windows/include)
    link_directories(../third_party/libcurl/windows/lib)
    add_executable(aliyun_assist_service ${DIR_SRCS}  ../VersionResource.rc)
    SET_TARGET_PROPERTIES(aliyun_assist_service PROPERTIES RUNTIME_OUTPUT_DIRECTORY_DEBUG   ${CMAKE_CURRENT_SOURCE_DIR}/../output/)
    SET_TARGET_PROPERTIES(aliyun_assist_service PROPERTIES RUNTIME_OUTPUT_DIRECTORY_RELEASE ${CMAKE_CURRENT_SOURCE_DIR}/../output/)
else(UNIX)
    include_directories(../third_party/libcurl/linux/include)
    link_directories(../third_party/libcurl/linux/lib)
	link_libraries(curl ssl crypto)
    include_directories(./linux)
	include_directories(./linux/xs)
    aux_source_directory(./linux DIR_SRCS)
	aux_source_directory(./linux/xs DIR_SRCS)
	add_executable(aliyun-service ${DIR_SRCS}  ../VersionResource.rc)
	set_target_properties(aliyun-service PROPERTIES RUNTIME_OUTPUT_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/../output/)

	#for static link ,only success on  Ubuntu 16.04.4 LTS
	if( DEFINED ENV{USE_STATIC_LINK} ) 
		set (CMAKE_C_FLAGS  "${CMAKE_C_FLAGS} -I../third_party/glibc/include")
		set (STATIC_GLIB_PATH "${CMAKE_CURRENT_SOURCE_DIR}/../third_party/glibc/lib")
		target_link_libraries(aliyun-service "-nostdlib -nostartfiles -static ${STATIC_GLIB_PATH}/crt1.o ${STATIC_GLIB_PATH}/crti.o  -Wl,--start-group ${STATIC_GLIB_PATH}/libpthread.a -lstdc++  ${STATIC_GLIB_PATH}/librt.a  ${STATIC_GLIB_PATH}/libc.a  -ldl -lgcc  -lgcc_eh -lm -Wl,--end-group  ${STATIC_GLIB_PATH}/crtn.o")
	else()
		target_link_libraries(aliyun-service  -lpthread -ldl -lrt -lm)
	endif()
endif()







