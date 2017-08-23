# Use this module by invoking find_package with the form:
#  find_package(PJSIP
#    [version] [EXACT]      # Minimum or EXACT version e.g. 1.9
#    [REQUIRED]             # Fail with error if PJSIP is not found
#    )
# This module finds headers and requested component libraries OR a CMake
# package configuration file provided by a "PJSIP CMake" build.  For the
# latter case skip to the "PJSIP CMake" section below.  For the former
# case results are reported in variables:
#  PJSIP_FOUND            - True if headers and requested libraries were found
#  PJSIP_INCLUDE_DIRS     - PJSIP include directories
#  PJSIP_LIBRARY_DIRS     - Link directories for PJSIP libraries
#  PJSIP_LIBRARIES        - PJSIP component libraries to be linked
#  PJSIP_VERSION          - PJSIP version
#
# This module reads hints about search locations from variables:
#  PJSIP_ROOT             - Preferred installation prefix
#   (or PJSIPROOT)
#
# You can set additional messages output with `set (PJSIP_DEBUG ON)`
#
#=============================================================================
# Copyright 2014 halex2005 <akharlov@gmail.com>
#
# Distributed under Apache v2.0 License (the "License");
# see accompanying file LICENSE for details.
#
# This software is distributed WITHOUT ANY WARRANTY; without even the
# implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
# See the License for more information.
#=============================================================================
# (To distribute this file outside of CMakeHelpers, substitute the full
#  License text for the above reference.)
#=============================================================================

#-------------------------------------------------------------------------------
#
# Setup environment
#
#-------------------------------------------------------------------------------
# If PJSIP_DIR is not set, look for PJSIPROOT and PJSIP_ROOT
# as alternatives, since these are more conventional for PJSIP.
if ("$ENV{PJSIP_DIR}" STREQUAL "")
    if (NOT "$ENV{PJSIP_ROOT}" STREQUAL "")
        set(ENV{PJSIP_DIR} $ENV{PJSIP_ROOT})
    elseif (NOT "$ENV{PJSIPROOT}" STREQUAL "")
        set(ENV{PJSIP_DIR} $ENV{PJSIPROOT})
    endif()
endif()
mark_as_advanced(PJSIP_DIR)
set(PJSIP_ROOT_DIR $ENV{PJSIP_DIR})

if (NOT DEFINED PJSIP_REGENERATE_PKGCONFIG)
    set (PJSIP_REGENERATE_PKGCONFIG OFF)
endif()

if (WIN32)
    if (NOT DEFINED PJSIP_USE_BUILD_FROM_SOURCE)
        set (PJSIP_USE_BUILD_FROM_SOURCE ON)
    endif()
    if (NOT DEFINED PJSIP_USE_PKGCONFIG)
        set (PJSIP_USE_PKGCONFIG OFF)
    endif()
    if (NOT DEFINED PJSIP_USE_STATIC_RUNTIME)
        set (PJSIP_USE_STATIC_RUNTIME OFF)
    endif()
    add_definitions(-DPJ_WIN32=1)
endif()

if (UNIX)
    if (NOT DEFINED PJSIP_USE_PKGCONFIG)
        set (PJSIP_USE_PKGCONFIG ON)
    endif()
    if (NOT DEFINED PJSIP_USE_BUILD_FROM_SOURCE)
        set (PJSIP_USE_BUILD_FROM_SOURCE OFF)
    endif()
endif()


#-------------------------------------------------------------------------------
#
# Get version
#
#-------------------------------------------------------------------------------
if(PJSIP_USE_BUILD_FROM_SOURCE)
    # try to get actual version from PJSIP source directory
    if (EXISTS ${PJSIP_ROOT_DIR}/version.mak)
        file(STRINGS "${PJSIP_ROOT_DIR}/version.mak" PJSIP_VERSION_STRINGS)
        FOREACH(i IN LISTS PJSIP_VERSION_STRINGS)
            string(REGEX REPLACE "export PJ_VERSION_MAJOR.*:=.*([0-9]+)" "\\1" _PJSIP_TEMP_ "${i}")
            if (NOT "${_PJSIP_TEMP_}" STREQUAL "${i}")
                set (PJSIP_VERSION_MAJOR ${_PJSIP_TEMP_})
            endif()
            string(REGEX REPLACE "export PJ_VERSION_MINOR.*:=.*([0-9]+)" "\\1" _PJSIP_TEMP_ "${i}")
            if (NOT "${_PJSIP_TEMP_}" STREQUAL "${i}")
                set (PJSIP_VERSION_MINOR ${_PJSIP_TEMP_})
            endif()
            string(REGEX REPLACE "export PJ_VERSION_REV.*:=.*([0-9]+)" "\\1" _PJSIP_TEMP_ "${i}")
            if (NOT "${_PJSIP_TEMP_}" STREQUAL "${i}")
                set (PJSIP_VERSION_REV ${_PJSIP_TEMP_})
            endif()
        ENDFOREACH()

        set (PJSIP_VERSION_SHORT "${PJSIP_VERSION_MAJOR}.${PJSIP_VERSION_MINOR}")
        set (PJSIP_VERSION "${PJSIP_VERSION_SHORT}.${PJSIP_VERSION_REV}")
        if (PJSIP_DEBUG)
            message(STATUS "found version ${PJSIP_VERSION} from ${PJSIP_ROOT_DIR}/version.mak")
        endif()
    else()
        set(PJSIP_VERSION "0.0.0")
        if (PJSIP_DEBUG)
            message(STATUS "there is no file ${PJSIP_ROOT_DIR}/version.mak, version set to ${PJSIP_VERSION}")
        endif()
    endif()

    if (PJSIP_FIND_VERSION_EXACT)
        if (NOT ${PJSIP_VERSION} VERSION_EQUAL ${PJSIP_FIND_VERSION})
            message(SEND_ERROR "Required exact version ${PJSIP_FIND_VERSION} not found (found version ${PJSIP_VERSION})")
            return()
        endif()
    else()
        if(PJSIP_FIND_VERSION)
            if (${PJSIP_VERSION} VERSION_LESS ${PJSIP_FIND_VERSION})
                message(SEND_ERROR "Required version ${PJSIP_FIND_VERSION} or greater not found (found version ${PJSIP_VERSION})")
                return()
            endif()
        else()
            # Any version is acceptable.
        endif()
    endif()

    if (PJSIP_USE_STATIC_RUNTIME)
        set(PJSIP_CRT_LINKAGE Static)
    else()
        set(PJSIP_CRT_LINKAGE Dynamic)
    endif()
endif()


#-------------------------------------------------------------------------------
#
# Discover PJSIP
#
#-------------------------------------------------------------------------------
if (PJSIP_USE_PKGCONFIG)
    if (PJSIP_DEBUG)
        message(STATUS "using PkgConfig to discover pjsip")
    endif()

    if (NOT $ENV{PKG_CONFIG_PATH} STREQUAL "" AND IS_DIRECTORY $ENV{PKG_CONFIG_PATH})
        # environment variable PKG_CONFIG_PATH exists and it is directory
        string(REPLACE "\\" "/" PJSIP_ROOT_DIR_FOR_PKGCONFIG ${PJSIP_ROOT_DIR})
        if (NOT EXISTS "$ENV{PKG_CONFIG_PATH}/libpjproject.pc" OR PJSIP_REGENERATE_PKGCONFIG)
            message (STATUS "generate $ENV{PKG_CONFIG_PATH}/libpjproject.pc")
            configure_file(${CMAKE_CURRENT_LIST_DIR}/libpjproject.pc.in "$ENV{PKG_CONFIG_PATH}/libpjproject.pc" @ONLY)
        else()
            message (STATUS "found $ENV{PKG_CONFIG_PATH}/libpjproject.pc, no regenerate")
        endif()
    else()
        message (STATUS "there is no PKG_CONFIG_PATH environment variable, no libpjproject.pc file generated")
    endif()

    find_package(PkgConfig REQUIRED)
    if (NOT PKG_CONFIG_FOUND)
        message(SEND_ERROR "PkgConfig not found")
        return()
    endif()

    if (PJSIP_FIND_VERSION)
        if (PJSIP_FIND_VERSION_EXACT)
            pkg_check_modules(PJSIP libpjproject=${PJSIP_FIND_VERSION} REQUIRED)
        else()
            pkg_check_modules(PJSIP libpjproject>=${PJSIP_FIND_VERSION} REQUIRED)
        endif()
    else()
        pkg_check_modules(PJSIP libpjproject REQUIRED)
    endif()
elseif(PJSIP_USE_BUILD_FROM_SOURCE)
    if (PJSIP_DEBUG)
        message(STATUS "not using PkgConfig to discover pjsip")
    endif()
    if( CMAKE_SIZEOF_VOID_P EQUAL 8 )
        set(PJSIP_TargetCPU "x86_64")
        add_definitions(-DPJ_M_X86_64)
    else()
        set(PJSIP_TargetCPU "i386")
        add_definitions(-DPJ_M_I386)
    endif()

    find_path(PJSIP_PJLIB_INCLUDE_DIR      NAMES "pjlib.h"      PATHS "${PJSIP_ROOT_DIR}/pjlib/include")
    find_path(PJSIP_PJLIB_UTIL_INCLUDE_DIR NAMES "pjlib-util.h" PATHS "${PJSIP_ROOT_DIR}/pjlib-util/include")
    find_path(PJSIP_PJMEDIA_INCLUDE_DIR    NAMES "pjmedia.h"    PATHS "${PJSIP_ROOT_DIR}/pjmedia/include")
    find_path(PJSIP_PJNATH_INCLUDE_DIR     NAMES "pjnath.h"     PATHS "${PJSIP_ROOT_DIR}/pjnath/include")
    find_path(PJSIP_PJSIP_INCLUDE_DIR      NAMES "pjsip.h"      PATHS "${PJSIP_ROOT_DIR}/pjsip/include")

    set (PJSIP_INCLUDE_DIR
        ${PJSIP_PJLIB_INCLUDE_DIR}
        ${PJSIP_PJLIB_UTIL_INCLUDE_DIR}
        ${PJSIP_PJMEDIA_INCLUDE_DIR}
        ${PJSIP_PJNATH_INCLUDE_DIR}
        ${PJSIP_PJSIP_INCLUDE_DIR}
    )

    if (WIN32)
        set (PJSIP_LIBRARY_DIR "${PJSIP_ROOT_DIR}/lib")
        if (PJSIP_USE_STATIC_RUNTIME)
            set (PJSIP_LIBRARIES "libpjproject-\$(Platform)-\$(PlatformToolset)-\$(Configuration)-Static.lib")
        else()
            set (PJSIP_LIBRARIES "libpjproject-\$(Platform)-\$(PlatformToolset)-\$(Configuration)-Dynamic.lib")
        endif()
        set (PJSIP_STATIC_LIBRARIES ${PJSIP_LIBRARIES} Ws2_32.lib)
    elseif(UNIX)
        set (PJSIP_LIBRARY_DIR /usr/lib/${PJSIP_TargetCPU}-unknown-linux-gnu)
        set (PJSIP_LIBRARIES
            pjsua2-${PJSIP_TargetCPU}-unknown-linux-gnu
            pjsua-${PJSIP_TargetCPU}-unknown-linux-gnu
            pjsip-ua-${PJSIP_TargetCPU}-unknown-linux-gnu
            pjsip-simple-${PJSIP_TargetCPU}-unknown-linux-gnu
            pjsip-${PJSIP_TargetCPU}-unknown-linux-gnu
            pjmedia-codec-${PJSIP_TargetCPU}-unknown-linux-gnu
            pjmedia-${PJSIP_TargetCPU}-unknown-linux-gnu
            pjmedia-videodev-${PJSIP_TargetCPU}-unknown-linux-gnu
            pjnath-${PJSIP_TargetCPU}-unknown-linux-gnu
            pjlib-util-${PJSIP_TargetCPU}-unknown-linux-gnu
            srtp-${PJSIP_TargetCPU}-unknown-linux-gnu
            resample-${PJSIP_TargetCPU}-unknown-linux-gnu
            gsmcodec-${PJSIP_TargetCPU}-unknown-linux-gnu
            speex-${PJSIP_TargetCPU}-unknown-linux-gnu
            ilbcodec-${PJSIP_TargetCPU}-unknown-linux-gnu
            g7221codec-${PJSIP_TargetCPU}-unknown-linux-gnu
            portaudio-${PJSIP_TargetCPU}-unknown-linux-gnu
            pj-${PJSIP_TargetCPU}-unknown-linux-gnu
        )
        set (PJSIP_STATIC_LIBRARIES
            ${PJSIP_LIBRARIES}
            stdc++
            m
            rt
            pthread
            asound
            SDL2
            avformat
            avcodec
            swscale
            avutil
            v4l2
            crypto
            ssl
            opencore-amrnb
            opencore-amrwb
            opencore-amrwbenc
        )
    endif()

    set (PJSIP_INCLUDE_DIRS ${PJSIP_INCLUDE_DIR})
    set (PJSIP_LIBRARY_DIRS ${PJSIP_LIBRARY_DIR})

    if ("${PJSIP_INCLUDE_DIRS}" MATCHES "NOTFOUND" OR "${PJSIP_LIBRARY_DIRS}" MATCHES "NOTFOUND")
        set(PJSIP_FOUND 0)
    else()
        set (PJSIP_FOUND 1)
    endif()
endif()

if (PJSIP_DEBUG)
    message(STATUS "PJSIP_ROOT_DIR     = ${PJSIP_ROOT_DIR}")
    message(STATUS "PJSIP_FOUND        = ${PJSIP_FOUND}")
    message(STATUS "PJSIP_INCLUDE_DIRS = ${PJSIP_INCLUDE_DIRS}")
    message(STATUS "PJSIP_LIBRARY_DIRS = ${PJSIP_LIBRARY_DIRS}")
    message(STATUS "PJSIP_LIBRARIES    = ${PJSIP_LIBRARIES}")
    message(STATUS "PJSIP_VERSION      = ${PJSIP_VERSION}")
endif()

if(PJSIP_FOUND)
else()
    if(PJSIP_FIND_REQUIRED)
        message(SEND_ERROR "Unable to find the requested PJSIP libraries.\n${PJSIP_ERROR_REASON}")
    else()
        if(NOT PJSIP_FIND_QUIETLY)
            # we opt not to automatically output PJSIP_ERROR_REASON here as
            # it could be quite lengthy and somewhat imposing in its requests
            # Since PJSIP is not always a required dependency we'll leave this
            # up to the end-user.
            if(PJSIP_DEBUG OR PJSIP_DETAILED_FAILURE_MSG)
                message(STATUS "Could NOT find PJSIP\n${PJSIP_ERROR_REASON}")
            else()
                message(STATUS "Could NOT find PJSIP")
            endif()
        endif()
    endif()
endif()