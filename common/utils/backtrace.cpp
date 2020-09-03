#include "backtrace.h"

#include <cstdio>
#include <cstdlib>
#include <memory>
#include <string>
#include <stdexcept>

#if !defined(_WIN32)
#include <cxxabi.h>

#define UNW_LOCAL_ONLY
#include "libunwind.h"
#endif

namespace {
template<typename ...Args>
std::string string_format(const char *format, Args... args) {
    // Extra space for '\0'
    size_t size = snprintf(nullptr, 0, format, args...) + 1;
    if (size <= 0){
        throw std::runtime_error( "Error during formatting." );
    }

    std::unique_ptr<char[]> buf(new char[size]);
    snprintf(buf.get(), size, format, args...);
    // We don't want the '\0' inside
    return std::string(buf.get(), buf.get() + size - 1);
}
}

void StackUnwind::generateUnwind() {
#if !defined(_WIN32)
    int retcode = 0;
    unw_cursor_t cursor;
    unw_context_t context;

    if (unw_getcontext(&context) != 0) {
        return;
    }
    if (unw_init_local(&cursor, &context) != 0) {
        return;
    }

    for (std::size_t i = 0; unw_step(&cursor) > 0 && i < kMaxStack; ++i) {
        if (i < kStackStart) {
            continue;
        }

        char mangName[256] = {'\0'};
        std::string offset_str = "";
        unw_word_t offset = 0;
        if (unw_get_proc_name(&cursor, mangName, sizeof(mangName), &offset) == 0) {
            offset_str = string_format("0x%x", offset);
        }

        unw_proc_info_t proc_info;
        std::string addr = "[]";
        if (unw_get_proc_info(&cursor, &proc_info) == 0) {
            addr = string_format("[0x%x]", proc_info.start_ip);
        }

        int status = 0;
        std::unique_ptr<char, void (*)(void*)> demangName(
            abi::__cxa_demangle(mangName, 0, 0, &status),
            ::free
        );
        // if demangling is successful, output the demangled function name
        if (status == 0) {
            // Success (see http://gcc.gnu.org/onlinedocs/libstdc++/libstdc++-html-USERS-4.3/a01696.html)
            StackUnwindEntry entry(i - 1, std::string(demangName.get()), offset_str, addr);
            m_stack.push_back(entry);
        } else {
            // Not successful - we will use mangled name
            StackUnwindEntry entry(i - 1, std::string(mangName), offset_str, addr);
            m_stack.push_back(entry);
        }
    }
#endif
}
