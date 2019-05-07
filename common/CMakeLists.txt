CMAKE_MINIMUM_REQUIRED(VERSION 3.1)
# CMake 最低版本号要求
PROJECT (alyun-assist) 

include_directories(../third_party)
INCLUDE_DIRECTORIES(../common)

if(WIN32)
    set(LIB_NAME assist-common)
    include_directories(../third_party/libcurl/windows/include)
    link_directories(../third_party/libcurl/windows/lib)
	add_definitions(-DUNICODE -D_UNICODE)
	link_libraries(libcurl)
    link_libraries(setupapi)
else(UNIX)
    INCLUDE_DIRECTORIES(/usr/include)
    LINK_DIRECTORIES(/usr/lib)
    LINK_DIRECTORIES(/usr/local/lib)
    include_directories(../third_party/libcurl/linux/include)
    link_directories(../third_party/libcurl/linux/lib)
	link_libraries(curl ssl crypto)
endif()

aux_source_directory(utils DIR_SRCS)
ADD_LIBRARY(assist-common ${DIR_SRCS})

