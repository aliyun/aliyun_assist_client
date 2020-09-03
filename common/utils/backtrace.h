#ifndef COMMON_UTILS_BACKTRACE_H_
#define COMMON_UTILS_BACKTRACE_H_
#include <iostream>
#include <string>
#include <vector>

class StackUnwind {
public:
    static const std::size_t kMaxStack = 64;
    static const std::size_t kStackStart = 2;  // We want to skip c'tor and StackUnwind::generateUnwind()

    class StackUnwindEntry {
    public:
        StackUnwindEntry(std::size_t index, const std::string &funcname,
            const std::string &offset, const std::string &addr)
            : m_index(index), m_funcname(funcname), m_offset(offset), m_addr(addr) {}

        std::size_t m_index;
        std::string m_funcname;
        std::string m_offset;
        std::string m_addr;

        friend std::ostream& operator<<(std::ostream& ss, const StackUnwindEntry& entry) {
           ss << "[" << entry.m_index << "] " << entry.m_funcname
                << (entry.m_offset.empty() ? "" : "+") << entry.m_offset << entry.m_addr;
           return ss;
        }

    private:
        StackUnwindEntry(void);
    };

    StackUnwind(void) {
        generateUnwind();
    }

    virtual ~StackUnwind(void) {
    }

    inline std::vector<StackUnwindEntry>& getLatestStack(void) {
        return m_stack;
    }

    friend inline std::ostream& operator<<(std::ostream& os, const StackUnwind& st) {
       std::vector<StackUnwindEntry>::const_iterator it = st.m_stack.begin();
       while (it != st.m_stack.end()) {
           os << "    " << *it++ << "\n";
       }
       return os;
    }

private:
    std::vector<StackUnwindEntry> m_stack;

    void generateUnwind();
};

#endif  // COMMON_UTILS_BACKTRACE_H_
